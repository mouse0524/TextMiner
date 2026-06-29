package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"textminer/pkg/logger"

	ocr "github.com/getcharzp/go-ocr"
	"github.com/up-zero/gotool/imageutil"
)

var (
	ocrInstance *OcrProcessor
	ocrOnce     sync.Once
	ocrInitErr  error
)

type OcrProcessor struct {
	engine ocr.Engine
}

func GetOcrProcessor() (*OcrProcessor, error) {
	ocrOnce.Do(func() {
		ocrInstance, ocrInitErr = NewOcrProcessor()
	})
	return ocrInstance, ocrInitErr
}

func NewOcrProcessor() (*OcrProcessor, error) {
	baseDir := filepath.Join(filepath.Dir(os.Args[0]), "models")
	libDir := filepath.Join(filepath.Dir(os.Args[0]), "lib")

	config := ocr.Config{
		OnnxRuntimeLibPath: filepath.Join(libDir, "onnxruntime.dll"),
		DetModelPath:       filepath.Join(baseDir, "det.onnx"),
		RecModelPath:       filepath.Join(baseDir, "rec.onnx"),
		DictPath:           filepath.Join(baseDir, "dict.txt"),
	}

	logger.Infof("初始化OCR引擎, 模型目录: %s", baseDir)

	engine, err := ocr.NewPaddleOcrEngine(config)
	if err != nil {
		logger.Errorf("创建OCR引擎失败: %v", err)
		return nil, fmt.Errorf("创建OCR引擎失败: %w", err)
	}

	logger.Info("OCR引擎初始化成功")

	return &OcrProcessor{
		engine: engine,
	}, nil
}

func (o *OcrProcessor) Recognize(imagePath string) (string, error) {
	img, err := imageutil.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("打开图片失败: %w", err)
	}

	boxes, err := o.engine.RunDetect(img)
	if err != nil {
		return "", fmt.Errorf("文字检测失败: %w", err)
	}

	var texts []string
	for _, box := range boxes {
		result, err := o.engine.RunRecognize(img, box)
		if err != nil {
			continue
		}
		if result.Text != "" {
			texts = append(texts, result.Text)
		}
	}

	if len(texts) == 0 {
		return "", nil
	}

	var result string
	for i, text := range texts {
		if i > 0 {
			result += "\n"
		}
		result += text
	}

	return result, nil
}

func (o *OcrProcessor) Close() {
	if o.engine != nil {
		o.engine.Destroy()
	}
}
