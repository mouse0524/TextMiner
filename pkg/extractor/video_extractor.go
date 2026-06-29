package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"textminer/pkg/logger"
)

type VideoExtractor struct{}

func NewVideoExtractor() (*VideoExtractor, error) {
	return &VideoExtractor{}, nil
}

func (e *VideoExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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

	logger.Infof("视频文件识别: %s, MIME类型: %s", filePath, mimeType)

	result := &ExtractResult{
		FileName: filepath.Base(filePath),
		FileType: mimeType,
		FileSize: fileSize,
		Status:   "success",
		Content:  "",
	}

	if !isFileAccessible(filePath) {
		logger.Warnf("视频文件不存在或无法访问: %s", filePath)
		result.Status = "failed"
		result.ErrorMessage = "文件不存在或无法访问"
		return result, fmt.Errorf("文件不存在或无法访问")
	}

	return result, nil
}
