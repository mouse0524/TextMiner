package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"textminer/pkg/logger"
)

type ImageExtractor struct {
	ocrProcessor *OcrProcessor
}

func NewImageExtractor() (*ImageExtractor, error) {
	ocrProcessor, err := GetOcrProcessor()
	if err != nil {
		logger.Errorf("获取OCR处理器失败: %v", err)
		return nil, fmt.Errorf("获取OCR处理器失败: %w", err)
	}

	return &ImageExtractor{
		ocrProcessor: ocrProcessor,
	}, nil
}

func (e *ImageExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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

	if !isFileAccessible(filePath) {
		logger.Warnf("图片文件不存在或无法访问: %s", filePath)
		result.Status = StatusFailed
		result.ErrorMessage = "文件不存在或无法访问"
		return result, fmt.Errorf("文件不存在或无法访问")
	}

	if !enableOcr {
		logger.Infof("图片OCR未启用: %s", filePath)
		result.Content = ""
		result.Status = StatusSuccess
		return result, nil
	}

	logger.Infof("开始图片OCR识别: %s", filePath)

	content, err := e.ocrProcessor.Recognize(filePath)
	if err != nil {
		logger.Errorf("图片OCR识别失败: %s, 错误: %v", filePath, err)
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("OCR识别失败: %v", err)
		return result, err
	}

	logger.Infof("图片OCR识别完成: %s, 识别长度: %d", filePath, len(content))

	result.Content = content
	return result, nil
}
