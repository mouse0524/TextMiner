package extractor

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type CsvExtractor struct{}

func (e *CsvExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}

	content, err := parseCsvContent(data)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("解析CSV失败: %v", err)
		return result, err
	}

	result.Content = content
	return result, nil
}

func parseCsvContent(data []byte) (string, error) {
	var csvReader *csv.Reader

	if isValidUTF8(data) {
		csvReader = csv.NewReader(bytes.NewReader(data))
	} else {
		gbkReader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		gbkData, err := io.ReadAll(gbkReader)
		if err != nil {
			return string(data), nil
		}
		csvReader = csv.NewReader(bytes.NewReader(gbkData))
	}

	csvReader.Comma = detectSeparator(string(data))
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return string(data), err
	}

	var result strings.Builder
	for i, row := range records {
		if len(row) > 0 {
			if i > 0 {
				result.WriteString("\n")
			}
			for j, field := range row {
				if j > 0 {
					result.WriteString("\t")
				}
				result.WriteString(field)
			}
		}
	}

	return result.String(), nil
}

func detectSeparator(content string) rune {
	commaCount := strings.Count(content, ",")
	semicolonCount := strings.Count(content, ";")
	tabCount := strings.Count(content, "\t")
	pipeCount := strings.Count(content, "|")

	maxCount := commaCount
	separator := ','

	if semicolonCount > maxCount {
		maxCount = semicolonCount
		separator = ';'
	}
	if tabCount > maxCount {
		maxCount = tabCount
		separator = '\t'
	}
	if pipeCount > maxCount {
		maxCount = pipeCount
		separator = '|'
	}

	return rune(separator)
}
