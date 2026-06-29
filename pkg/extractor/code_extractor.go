package extractor

import (
	"fmt"
	"os"
	"path/filepath"

	"textminer/pkg/logger"
)

// CodeExtractor 代码文件提取器
type CodeExtractor struct{}

// Extract 提取代码文件内容
func (e *CodeExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	if enableOcr {
		logger.Warnf("代码文件不支持 OCR: %s", filePath)
	}

	fileInfo, err := os.Stat(filePath)
	fileSize := int64(0)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	detector := GetFileTypeDetector()
	_, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil || mimeType == "" {
		mimeType = resolveMimeType(filePath)
	}

	result := &ExtractResult{
		FileName: filepath.Base(filePath),
		FileType: mimeType,
		FileSize: fileSize,
		Status:   StatusSuccess,
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}

	content, err := decodeText(data)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("解码失败: %v", err)
		return result, err
	}

	result.Content = content
	return result, nil
}
