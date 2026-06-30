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

	// BMP/TIFF 解码器需要在 image.Decode 前注册（Go 标准库未自带）。
	// 这两个库在 init() 期间会调用 image.RegisterFormat，
	// 后续 imageutil.Open / image.Decode 即可识别 .bmp/.tif/.tiff。
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
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
		// OCR 引擎对部分图片格式（HEIC/ICO/RAW 等）无解码能力时会失败。
		// 退化为元数据提取：保证 ExtractResult 至少返回格式/尺寸信息，
		// 状态置为 StatusSkipped 而非 failed，方便上层 DLP 区分"未识别"与"真实失败"。
		logger.Warnf("图片OCR识别失败: %s, 错误: %v, 降级为元数据提取", filePath, err)
		format, width, height, dimErr := readImageDimensions(filePath)
		var b strings.Builder
		fmt.Fprintf(&b, "[image: OCR unavailable for this format")
		if dimErr == nil {
			fmt.Fprintf(&b, ", %s, %dx%d", format, width, height)
		} else {
			fmt.Fprintf(&b, ", dimensions unavailable")
		}
		b.WriteString("]")
		result.Content = b.String()
		result.Status = StatusSkipped
		result.ErrorMessage = fmt.Sprintf("OCR不支持该图片格式: %v", err)
		return result, nil
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
