package extractor

import (
	"fmt"

	"textminer/pkg/logger"
)

// MimeOnlyExtractor 用于只需识别 MIME 类型、但不提取文本内容的文件类别
// （如音频、视频、可执行文件、压缩包中的二进制等）。
// 合并自原来的 AudioExtractor / VideoExtractor / MimeOnlyExtractor。
type MimeOnlyExtractor struct {
	kind string // "audio" | "video" | "mime"
}

func NewAudioExtractor() (*MimeOnlyExtractor, error)    { return &MimeOnlyExtractor{kind: "audio"}, nil }
func NewVideoExtractor() (*MimeOnlyExtractor, error)    { return &MimeOnlyExtractor{kind: "video"}, nil }
func NewMimeOnlyExtractor() (*MimeOnlyExtractor, error) { return &MimeOnlyExtractor{kind: "mime"}, nil }

// Extract 探测 MIME 类型并返回 skipped 状态（不提取内容）。
func (e *MimeOnlyExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	if enableOcr {
		logger.Warnf("%s 文件不支持 OCR: %s", e.kind, filePath)
	}

	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	kindLabel := map[string]string{
		"audio": "音频",
		"video": "视频",
		"mime":  "二进制",
	}[e.kind]
	if kindLabel == "" {
		kindLabel = e.kind
	}
	logger.Infof("%s文件识别: %s, MIME类型: %s", kindLabel, filePath, result.FileType)

	// 这类文件不提取文本内容，返回 skipped 状态而非误导性 success
	result.Status = StatusSkipped
	result.Skipped = true
	return result, nil
}
