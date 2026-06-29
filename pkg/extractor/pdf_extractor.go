package extractor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	fileInfo, err := os.Stat(filePath)
	fileSize := int64(0)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	detector := GetFileTypeDetector()
	_, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil || mimeType == "" {
		mimeType = MapExtensionToMimeType(ext[1:])
	}

	result := &ExtractResult{
		FileName: filepath.Base(filePath),
		FileType: mimeType,
		FileSize: fileSize,
		Status:   "success",
	}

	if !isFileAccessible(filePath) {
		result.Status = "failed"
		result.ErrorMessage = "文件不存在或无法访问"
		return result, fmt.Errorf("文件不存在或无法访问")
	}

	file, err := os.Open(filePath)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("打开PDF文件失败: %v", err)
		return result, err
	}
	defer file.Close()

	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("创建PDF读取器失败: %v", err)
		return result, err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("获取PDF页数失败: %v", err)
		return result, err
	}

	logger.Infof("PDF文档页数: %d", numPages)

	type pageResult struct {
		index int
		text  string
	}

	const batchSize = 100
	var builder strings.Builder
	builder.Grow(numPages * 1000)

	for startPage := 1; startPage <= numPages; startPage += batchSize {
		endPage := startPage + batchSize - 1
		if endPage > numPages {
			endPage = numPages
		}

		logger.Infof("处理批次: 页面 %d-%d / %d", startPage, endPage, numPages)

		batchSizeActual := endPage - startPage + 1
		results := make([]pageResult, batchSizeActual)
		var wg sync.WaitGroup
		var mu sync.Mutex

		pageChan := make(chan int, batchSizeActual)

		numWorkers := runtime.NumCPU()
		if batchSizeActual < numWorkers {
			numWorkers = batchSizeActual
		}

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pageNum := range pageChan {
					page, err := pdfReader.GetPage(pageNum)
					if err != nil {
						logger.Warnf("Failed to get page %d: %v", pageNum, err)
						continue
					}

					ex, err := extractor.New(page)
					if err != nil {
						logger.Warnf("Failed to create extractor for page %d: %v", pageNum, err)
						continue
					}

					text, err := ex.ExtractText()
					if err != nil {
						logger.Warnf("Failed to extract text from page %d: %v", pageNum, err)
						continue
					}

					mu.Lock()
					results[pageNum-startPage] = pageResult{index: pageNum, text: text}
					mu.Unlock()
				}
			}()
		}

		for i := startPage; i <= endPage; i++ {
			pageChan <- i
		}
		close(pageChan)

		wg.Wait()

		sort.Slice(results, func(i, j int) bool {
			return results[i].index < results[j].index
		})

		for _, res := range results {
			builder.WriteString(res.text)
		}

		logger.Infof("批次完成: 页面 %d-%d, 当前内容长度: %d", startPage, endPage, builder.Len())
	}

	result.Content = strings.TrimSpace(builder.String())
	if result.Content == "" {
		result.Status = "failed"
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
		return "", err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("获取PDF页数失败: %w", err)
	}

	pageNumbers, err := normalizePageRange(numPages, pageRange, false)
	if err != nil {
		return "", fmt.Errorf("页面范围解析失败: %w", err)
	}

	type pageResult struct {
		index int
		text  string
	}

	const batchSize = 100
	var builder strings.Builder
	builder.Grow(len(pageNumbers) * 1000)

	for batchIdx := 0; batchIdx < len(pageNumbers); batchIdx += batchSize {
		endIdx := batchIdx + batchSize
		if endIdx > len(pageNumbers) {
			endIdx = len(pageNumbers)
		}

		batchPages := pageNumbers[batchIdx:endIdx]
		logger.Infof("处理批次: 页面 %d-%d / %d", batchIdx+1, endIdx, len(pageNumbers))

		results := make([]pageResult, len(batchPages))
		var wg sync.WaitGroup
		var mu sync.Mutex

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
					page, err := pdfReader.GetPage(pageNum)
					if err != nil {
						logger.Warnf("Failed to get page %d: %v", pageNum, err)
						continue
					}

					ex, err := extractor.New(page)
					if err != nil {
						logger.Warnf("Failed to create extractor for page %d: %v", pageNum, err)
						continue
					}

					text, err := ex.ExtractText()
					if err != nil {
						logger.Warnf("Failed to extract text from page %d: %v", pageNum, err)
						continue
					}

					mu.Lock()
					for i, p := range batchPages {
						if p == pageNum {
							results[i] = pageResult{index: pageNum, text: text}
							break
						}
					}
					mu.Unlock()
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

	return strings.TrimSpace(builder.String()), nil
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
