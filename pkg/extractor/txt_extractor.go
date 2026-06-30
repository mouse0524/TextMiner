package extractor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"textminer/pkg/logger"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// TxtExtractor 同时处理纯文本与代码文件：唯一的差别是 isCode 时对 OCR 报警一次。
// 合并自原 TxtExtractor 与 CodeExtractor（2026-06-30 第五轮审查）。
type TxtExtractor struct {
	isCode bool
}

// NewTxtExtractor isCode=false：纯文本/log/ini。
// NewTxtExtractor isCode=true：源代码/markup 文件。
func NewTxtExtractor(isCode bool) *TxtExtractor {
	return &TxtExtractor{isCode: isCode}
}

func (e *TxtExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	if e.isCode && enableOcr {
		logger.Warnf("代码文件不支持 OCR: %s", filePath)
	}

	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	// 流式读取：64KB bufio 缓冲，避免 os.ReadFile 一次性分配整个文件
	f, err := os.Open(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}
	defer f.Close()

	br := getBufioReader(f)
	defer putBufioReader(br)
	data, err := io.ReadAll(br)
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

func decodeText(data []byte) (string, error) {
	if isValidUTF8(data) {
		return string(data), nil
	}

	gbkReader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	gbkData, err := io.ReadAll(gbkReader)
	if err == nil && len(gbkData) > 0 {
		return string(gbkData), nil
	}

	gb18030Reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	gb18030Data, err := io.ReadAll(gb18030Reader)
	if err == nil && len(gb18030Data) > 0 {
		return string(gb18030Data), nil
	}

	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16LE(data[2:]), nil
	}

	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BE(data[2:]), nil
	}

	return string(data), nil
}

func isValidUTF8(data []byte) bool {
	return utf8.Valid(data)
}

func decodeUTF16LE(data []byte) string {
	var result strings.Builder
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			result.WriteRune(rune(data[i]) | rune(data[i+1])<<8)
		}
	}
	return result.String()
}

func decodeUTF16BE(data []byte) string {
	var result strings.Builder
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			result.WriteRune(rune(data[i])<<8 | rune(data[i+1]))
		}
	}
	return result.String()
}
