package extractor

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
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
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	if !enableOcr {
		logger.Infof("图片OCR未启用，提取基础元数据: %s", filePath)
		format, width, height, dimErr := readImageDimensions(filePath)
		var b strings.Builder
		fmt.Fprintf(&b, "[image: no OCR enabled")
		if dimErr == nil {
			fmt.Fprintf(&b, ", %s, %dx%d", format, width, height)
		} else {
			fmt.Fprintf(&b, ", dimensions unavailable")
		}
		b.WriteString("]")
		result.Content = b.String()
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

// readImageDimensions 读取图片的格式与像素尺寸，不做完整解码。
// 返回 (format, width, height, err)；format 为 png/jpeg/gif 等小写字符串。
func readImageDimensions(filePath string) (string, int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, 0, err
	}
	defer f.Close()

	var head [512]byte
	n, _ := f.Read(head[:])
	cfg, _, err := image.DecodeConfig(bytes.NewReader(head[:n]))
	if err != nil {
		return "", 0, 0, err
	}
	ext := ""
	if dot := strings.LastIndex(filePath, "."); dot >= 0 {
		ext = strings.ToLower(filePath[dot+1:])
	}
	return ext, cfg.Width, cfg.Height, nil
}
