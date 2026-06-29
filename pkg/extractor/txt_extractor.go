package extractor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type TxtExtractor struct{}

func (e *TxtExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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
