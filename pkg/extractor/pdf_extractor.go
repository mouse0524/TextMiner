package extractor

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"textminer/pkg/logger"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type PdfExtractor struct{}

func (e *PdfExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	file, err := os.Open(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("打开PDF文件失败: %v", err)
		return result, err
	}
	defer file.Close()

	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		// 加密 PDF：unipdf v3 在 NewPdfReader 阶段可能就返回错误。
		// 错误信息含 "encrypted"/"password" 时归类为加密场景，返回 ErrEncrypted 哨兵
		// 让调用方能识别为 StatusEncrypted 而非通用失败。
		if isEncryptedPdfError(err) {
			result.Status = StatusEncrypted
			result.IsEncrypt = 1
			result.ErrorMessage = "PDF 已加密，无法提取明文内容"
			return result, ErrEncrypted
		}
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("创建PDF读取器失败: %v", err)
		return result, err
	}

	// 即使 NewPdfReader 成功，某些 PDF 也可能标注为加密
	if isEnc, _ := pdfReader.IsEncrypted(); isEnc {
		result.Status = StatusEncrypted
		result.IsEncrypt = 1
		result.ErrorMessage = "PDF 已加密，无法提取明文内容"
		return result, ErrEncrypted
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("获取PDF页数失败: %v", err)
		return result, err
	}

	logger.Infof("PDF文档页数: %d", numPages)

	pages := make([]int, numPages)
	for i := range pages {
		pages[i] = i + 1
	}
	content, err := processPdfBatch(pdfReader, pages)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("提取PDF文本失败: %v", err)
		return result, err
	}

	result.Content = strings.TrimSpace(content)
	if result.Content == "" {
		result.Status = StatusFailed
		result.ErrorMessage = "未提取到文本内容"
		return result, fmt.Errorf("未提取到文本内容")
	}

	logger.Infof("PDF提取完成: 文件=%s, 内容长度=%d", filePath, len(result.Content))

	return result, nil
}

func ExtractPdfText(filePath string, pageRange string, password string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		if isEncryptedPdfError(err) {
			return "", ErrEncrypted
		}
		return "", err
	}

	// 加密 PDF：若调用方提供了 password，尝试用 Decrypt 解密。
	// 解密失败仍按加密处理，避免泄露部分明文。
	if isEnc, _ := pdfReader.IsEncrypted(); isEnc {
		if password == "" {
			return "", ErrEncrypted
		}
		if _, derr := pdfReader.Decrypt([]byte(password)); derr != nil {
			return "", ErrEncrypted
		}
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("获取PDF页数失败: %w", err)
	}

	pageNumbers, err := normalizePageRange(numPages, pageRange, false)
	if err != nil {
		return "", fmt.Errorf("页面范围解析失败: %w", err)
	}

	content, err := processPdfBatch(pdfReader, pageNumbers)
	if err != nil {
		return "", fmt.Errorf("提取PDF文本失败: %w", err)
	}
	return strings.TrimSpace(content), nil
}

// processPdfBatch 给定 pdf reader 和 page 列表，并发提取所有页文本后按 page 顺序拼接。
// 公共逻辑：原 PdfExtractor.Extract 和 ExtractPdfText 各自重复了 ~80 行的 batch-worker-sort-join
// 模板，仅在「page 列表是否连续」上不同——这里统一以稀疏 page 列表 + pageIdx map 处理，两侧零修改。
// batch=100 限制单批内存峰值；worker 数 = min(NumCPU, batchSize)；错误降级为 logger.Warnf
// 保留 best-effort 行为（与原代码一致）。
const pdfBatchSize = 100

type pdfPageResult struct {
	index int
	text  string
}

func processPdfBatch(pdfReader *model.PdfReader, pageNumbers []int) (string, error) {
	if len(pageNumbers) == 0 {
		return "", nil
	}

	var builder strings.Builder
	builder.Grow(len(pageNumbers) * 1000)

	for batchIdx := 0; batchIdx < len(pageNumbers); batchIdx += pdfBatchSize {
		endIdx := batchIdx + pdfBatchSize
		if endIdx > len(pageNumbers) {
			endIdx = len(pageNumbers)
		}
		batchPages := pageNumbers[batchIdx:endIdx]

		logger.Infof("处理批次: 页面 %d-%d / %d", batchIdx+1, endIdx, len(pageNumbers))

		results := make([]pdfPageResult, len(batchPages))
		// 用 map 把 pageNum 映射到 results 下标，避免 O(M) 线性扫描
		pageIdx := make(map[int]int, len(batchPages))
		for i, p := range batchPages {
			pageIdx[p] = i
		}
		var wg sync.WaitGroup

		pageChan := make(chan int, len(batchPages))
		numWorkers := runtime.NumCPU()
		if len(batchPages) < numWorkers {
			numWorkers = len(batchPages)
		}

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pageNum := range pageChan {
					text, err := extractSinglePage(pdfReader, pageNum)
					if err != nil || text == "" {
						continue
					}
					if i, ok := pageIdx[pageNum]; ok {
						results[i] = pdfPageResult{index: pageNum, text: text}
					}
				}
			}()
		}

		for _, pageNum := range batchPages {
			pageChan <- pageNum
		}
		close(pageChan)
		wg.Wait()

		sort.Slice(results, func(i, j int) bool {
			return results[i].index < results[j].index
		})

		for _, res := range results {
			builder.WriteString(res.text)
		}
		logger.Infof("批次完成: 页面 %d-%d, 当前内容长度: %d", batchIdx+1, endIdx, builder.Len())
	}

	return builder.String(), nil
}

// extractSinglePage 提取单页文本：失败/为空返回 ("", err)。err 不为 nil 时同样写 warn 日志。
func extractSinglePage(pdfReader *model.PdfReader, pageNum int) (string, error) {
	page, err := pdfReader.GetPage(pageNum)
	if err != nil {
		logger.Warnf("Failed to get page %d: %v", pageNum, err)
		return "", err
	}
	ex, err := extractor.New(page)
	if err != nil {
		logger.Warnf("Failed to create extractor for page %d: %v", pageNum, err)
		return "", err
	}
	text, err := ex.ExtractText()
	if err != nil {
		logger.Warnf("Failed to extract text from page %d: %v", pageNum, err)
		return "", err
	}
	return text, nil
}

// isEncryptedPdfError 判断 unipdf 在 NewPdfReader 阶段返回的错误是否表明 PDF 已加密。
// unipdf v3 没有暴露 ErrEncrypted 哨兵，靠错误信息字符串判定。
func isEncryptedPdfError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "encrypted") ||
		strings.Contains(msg, "password")
}

func GetPdfPageCount(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		return 0, err
	}

	return pdfReader.GetNumPages()
}

func normalizePageRange(pageCount int, pageRange string, ignoreInvalidPages bool) ([]int, error) {
	if pageRange == "" {
		pageRange = "first-last"
	}

	var pageNumbers []int
	seenPages := make(map[int]bool)

	pageRanges := strings.Split(pageRange, ",")
	for _, rangeStr := range pageRanges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}

		parts := strings.Split(rangeStr, "-")
		if len(parts) == 0 || len(parts) > 2 {
			return nil, fmt.Errorf("invalid page range: %s", rangeStr)
		}

		var start, end int
		var err error

		if len(parts) == 1 {
			start, err = parsePageNumber(parts[0], pageCount)
			if err != nil {
				if ignoreInvalidPages {
					continue
				}
				return nil, err
			}
			end = start
		} else {
			start, err = parsePageNumber(parts[0], pageCount)
			if err != nil {
				if ignoreInvalidPages {
					continue
				}
				return nil, err
			}

			end, err = parsePageNumber(parts[1], pageCount)
			if err != nil {
				if ignoreInvalidPages {
					continue
				}
				return nil, err
			}

			if start > end {
				start, end = end, start
			}
		}

		for i := start; i <= end; i++ {
			if i < 1 || i > pageCount {
				if ignoreInvalidPages {
					continue
				}
				return nil, fmt.Errorf("page %d is out of range (1-%d)", i, pageCount)
			}

			if !seenPages[i] {
				seenPages[i] = true
				pageNumbers = append(pageNumbers, i)
			}
		}
	}

	if len(pageNumbers) == 0 {
		return nil, errors.New("no valid pages in the specified range")
	}

	return pageNumbers, nil
}

func parsePageNumber(pageStr string, pageCount int) (int, error) {
	pageStr = strings.TrimSpace(pageStr)

	switch pageStr {
	case "first":
		return 1, nil
	case "last":
		return pageCount, nil
	}

	if strings.HasPrefix(pageStr, "r") {
		numStr := strings.TrimPrefix(pageStr, "r")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid reverse page number: %s", pageStr)
		}
		return pageCount - num, nil
	}

	num, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0, fmt.Errorf("invalid page number: %s", pageStr)
	}

	return num, nil
}
