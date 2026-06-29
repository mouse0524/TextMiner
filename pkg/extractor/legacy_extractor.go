package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"textminer/pkg/logger"

	goppt "github.com/KSpaceer/goppt"
	gocatdoc "github.com/semvis123/go-catdoc"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/shakinm/xlsReader/xls"
)

func extractWordDocContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开DOC文件失败: %w", err)
	}
	defer file.Close()

	text, err := gocatdoc.GetTextFromFile(file)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "File is encrypted") || strings.Contains(errMsg, "encrypted") {
			return "", ErrEncrypted
		}
		if strings.Contains(errMsg, "fast-saved") && text != "" {
			return text, nil
		}
		return "", fmt.Errorf("提取DOC内容失败: %w", err)
	}

	if text == "" {
		return "", fmt.Errorf("未找到DOC内容")
	}

	return strings.TrimSpace(text), nil
}

func extractExcelXlsContent(filePath string) (string, error) {
	workbook, err := xls.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("打开XLS文件失败: %w", err)
	}

	var content strings.Builder
	content.Grow(1024 * 1024)

	numSheets := workbook.GetNumberSheets()
	for i := 0; i < numSheets; i++ {
		sheet, err := workbook.GetSheet(i)
		if err != nil {
			continue
		}

		numRows := sheet.GetNumberRows()
		for j := 0; j <= numRows; j++ {
			row, err := sheet.GetRow(j)
			if err != nil {
				continue
			}

			rowContent := strings.Builder{}
			rowContent.Grow(256)

			cols := row.GetCols()
			for _, cell := range cols {
				cellType := cell.GetType()
				if cellType == "FakeBlank" {
					continue
				}

				xfIndex := cell.GetXFIndex()
				formatIndex := workbook.GetXFbyIndex(xfIndex)
				format := workbook.GetFormatByIndex(formatIndex.GetFormatIndex())
				cellValue := format.GetFormatString(cell)

				if cellValue == "" {
					cellValue = cell.GetString()
				}

				if cellValue != "" {
					if rowContent.Len() > 0 {
						rowContent.WriteString(" ")
					}
					rowContent.WriteString(cellValue)
				}
			}
			if rowContent.Len() > 0 {
				content.WriteString(rowContent.String())
				content.WriteString("\n")
			}
		}
	}

	result := strings.TrimSpace(content.String())
	if result == "" {
		return "", fmt.Errorf("未找到Excel内容")
	}

	return result, nil
}

func extractPowerPointPptContent(filePath string) (string, error) {
	return extractPptContent(filePath, false)
}

func extractPowerPointPptContentWithMmap(filePath string) (string, error) {
	return extractPptContent(filePath, true)
}

func extractPptContent(filePath string, useMmap bool) (string, error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", fmt.Errorf("打开PPT文件失败: %w", openErr)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	method := ""
	if useMmap {
		method = "(Mmap版)"
	}
	logger.Infof("开始提取PPT文件%s: %s, 大小: %d bytes", method, filePath, fileSize)

	if fileSize > 100*1024*1024 {
		logger.Warnf("PPT文件较大: %d bytes，可能需要较长时间", fileSize)
	}

	var text string
	var err error
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("PPT提取发生panic: %v", r)
			if text != "" {
				text = strings.TrimSpace(text)
			}
		}
	}()

	if useMmap {
		text, err = goppt.ExtractTextWithMmap(file)
	} else {
		text, err = goppt.ExtractText(file)
	}

	if err != nil {
		return handlePptExtractionError(filePath, text, err)
	}

	if text == "" {
		return "", fmt.Errorf("未找到PPT内容")
	}

	logger.Infof("PPT提取完成%s: %s, 内容长度: %d", method, filePath, len(text))
	return strings.TrimSpace(text), nil
}

func handlePptExtractionError(filePath string, text string, err error) (string, error) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "EOF") || strings.Contains(errMsg, "unexpected EOF") {
		if text != "" {
			logger.Warnf("PPT文件可能不完整，返回已提取的内容: %s, 长度: %d", filePath, len(text))
			return strings.TrimSpace(text), nil
		}
		logger.Warnf("PPT文件提取遇到EOF错误且无内容: %s", filePath)
		return "", fmt.Errorf("PPT文件提取失败: 未提取到内容")
	}
	if strings.Contains(errMsg, "mismatch record type") {
		logger.Warnf("PPT文件可能已加密，返回错误: %s", filePath)
		return "", ErrEncrypted
	}
	logger.Errorf("提取PPT内容失败: %v", err)
	return "", fmt.Errorf("提取PPT内容失败: %w", err)
}

var xlsContentCache *lru.Cache[string, string]

func init() {
	xlsContentCache, _ = lru.New[string, string](128)
}

func extractExcelXlsContentWithCache(filePath string) (string, error) {
	return extractWithCache(xlsContentCache, filePath, extractExcelXlsContent)
}

func ClearXLSCache() {
	clearCache(xlsContentCache)
}

func GetXLSCacheSize() int {
	return getCacheSize(xlsContentCache)
}

var pptContentCache *lru.Cache[string, string]

func init() {
	pptContentCache, _ = lru.New[string, string](128)
}

func extractPowerPointPptContentWithCache(filePath string) (string, error) {
	return extractWithCache(pptContentCache, filePath, extractPowerPointPptContentWithMmap)
}

func ClearPPTCache() {
	clearCache(pptContentCache)
}

func GetPPTCacheSize() int {
	return getCacheSize(pptContentCache)
}

type WordDocumentExtractor struct{}

func (e *WordDocumentExtractor) Extract(filePath string) (*ExtractResult, error) {
	return extractWithFileCheck(filePath, extractWordDocContent)
}

func extractExcelXlsbContent(filePath string) (string, error) {
	parser := NewXlsbParser()
	content, err := parser.Parse(filePath)
	if err != nil {
		return "", fmt.Errorf("解析XLSB文件失败: %v", err)
	}

	if content == "" {
		return "", fmt.Errorf("未找到XLSB内容")
	}

	return strings.TrimSpace(content), nil
}

type ExcelXlsExtractor struct{}

func (e *ExcelXlsExtractor) Extract(filePath string) (*ExtractResult, error) {
	return extractWithFileCheck(filePath, extractExcelXlsContentWithCache)
}

type PowerPointPptExtractor struct{}

func (e *PowerPointPptExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	// 先检查文件是否加密
	detector := NewEncryptionDetector()
	isEnc := detector.CheckEncryption(filePath)
	if isEnc == 1 {
		logger.Infof("PPT文件已加密，直接返回: 文件=%s", filePath)
		return &ExtractResult{
			FileName:     filepath.Base(filePath),
			FileType:     "application/vnd.ms-powerpoint",
			FileSize:     0,
			Status:       StatusFailed,
			Content:      "",
			ErrorMessage: "文件已加密，无法提取内容",
			IsEncrypt:    1,
			ExecuteTime:  "0.0000",
		}, ErrEncrypted
	}

	return extractWithFileCheck(filePath, extractPowerPointPptContentWithCache)
}
