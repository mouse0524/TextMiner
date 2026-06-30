package extractor

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"textminer/pkg/logger"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type extractFunc func(filePath string) (string, error)

type ExtractContext struct {
	FileName string
	FileType string
	FileSize int64
	FilePath string
	Ext      string
	MimeType string
	// DetectedFileType / DetectedMime 是 GetDetailedInfo 一次性计算的缓存结果，
	// 避免 extractFileDetect 等下游再调一次 GetDetailedInfo（Magika ONNX 推理昂贵）。
	DetectedFileType string
	DetectedMime     string
}

// MaxCacheEntryBytes 超过此大小的 extracted content 不入 LRU 缓存。
// 16MB 阈值：典型文档（PDF/PPTX/DOCX）解压后文本远低于此值；
// 巨型压缩包/日志超过此值时直接走非缓存路径，避免撑爆 LRU。
const MaxCacheEntryBytes = 16 << 20

// cacheKeyContentBytes 参与 SHA-1 计算的字节数：64KB 头足以检测"同长度但内容不同"。
const cacheKeyContentBytes = 64 << 10

func prepareExtractContext(filePath string) (*ExtractContext, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	return prepareExtractContextWithInfo(filePath, info)
}

// prepareExtractContextWithInfo 用已知的 os.FileInfo 构造 ExtractContext，
// 避免在已 Stat 的代码路径上重复系统调用。FileSize 来自 info。
// 同时缓存 GetDetailedInfo 结果到 ctx.DetectedFileType / DetectedMime，
// 下游 extractFileDetect 复用而不再调用。
func prepareExtractContextWithInfo(filePath string, info os.FileInfo) (*ExtractContext, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	detector := GetFileTypeDetector()
	detectedType, detectedMime, err := detector.GetDetailedInfo(filePath)
	if err != nil || detectedMime == "" {
		// fallback：使用扩展名推断 mime
		detectedType = strings.TrimPrefix(ext, ".")
		detectedMime = MapExtensionToMimeType(detectedType)
	}

	mimeType := detectedMime
	if mimeType == "" {
		mimeType = MapExtensionToMimeType(strings.TrimPrefix(ext, "."))
	}

	return &ExtractContext{
		FileName:         filepath.Base(filePath),
		FileType:         mimeType,
		FileSize:         info.Size(),
		FilePath:         filePath,
		Ext:              ext,
		MimeType:         mimeType,
		DetectedFileType: detectedType,
		DetectedMime:     detectedMime,
	}, nil
}

func newSuccessResult(ctx *ExtractContext, content string) *ExtractResult {
	return &ExtractResult{
		FileName: ctx.FileName,
		FileType: ctx.FileType,
		FileSize: ctx.FileSize,
		Status:   StatusSuccess,
		Content:  content,
	}
}

func newErrorResult(ctx *ExtractContext, errMsg string) *ExtractResult {
	return &ExtractResult{
		FileName:     ctx.FileName,
		FileType:     ctx.FileType,
		FileSize:     ctx.FileSize,
		Status:       StatusFailed,
		ErrorMessage: errMsg,
	}
}

func newFileAccessErrorResult(filePath string) *ExtractResult {
	ext := strings.ToLower(filepath.Ext(filePath))
	detector := GetFileTypeDetector()
	_, mimeType, _ := detector.GetDetailedInfo(filePath)
	if mimeType == "" {
		mimeType = MapExtensionToMimeType(strings.TrimPrefix(ext, "."))
	}

	return &ExtractResult{
		FileName:     filepath.Base(filePath),
		FileType:     mimeType,
		FileSize:     0,
		Status:       StatusFailed,
		ErrorMessage: "文件不存在或无法访问",
	}
}

func extractWithCache(cache *lru.Cache[string, string], filePath string, extractor extractFunc) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := buildCacheKey(filePath, fileInfo.ModTime().UnixNano(), fileInfo.Size(), false)
	if cached, ok := cache.Get(cacheKey); ok {
		logger.Infof("使用缓存内容: %s", filePath)
		return cached, nil
	}

	startTime := time.Now()
	content, err := extractor(filePath)
	if err != nil {
		return "", err
	}
	elapsed := time.Since(startTime)
	logger.Infof("提取完成: %s, 耗时: %v, 内容长度: %d", filePath, elapsed, len(content))

	// value size cap：超大内容不入缓存
	if len(content) <= MaxCacheEntryBytes {
		cache.Add(cacheKey, content)
	} else {
		logger.Warnf("内容长度 %d > MaxCacheEntryBytes %d，跳过缓存: %s", len(content), MaxCacheEntryBytes, filePath)
	}
	return content, nil
}

// buildCacheKey 高效构造 cache key：path + mtime + size + ocr + 头 64KB SHA-1。
// enableOcr 参与 key 计算：OCR 与非 OCR 提取结果不共享缓存，避免 OCR 模式命中
// 非 OCR 缓存导致 OCR 文本缺失（数据丢失）。
// 哈希避免"同长度但内容不同"误命中：例如 touch 之后覆盖同大小文件。
// 仅在 key 构造时读前 64KB，开销远低于 io.ReadAll 整个文件。
func buildCacheKey(path string, mtimeUnixNano, size int64, enableOcr bool) string {
	h := sha1.New()
	f, err := os.Open(path)
	if err == nil {
		_, _ = io.CopyN(h, f, cacheKeyContentBytes)
		_ = f.Close()
	}
	sum := h.Sum(nil)

	var b strings.Builder
	b.Grow(len(path) + 80)
	b.WriteString(path)
	b.WriteByte(':')
	b.WriteString(strconv.FormatInt(mtimeUnixNano, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatInt(size, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatBool(enableOcr))
	b.WriteByte(':')
	// 仅写 8 字节哈希前缀（碰撞概率 2^-32），节省缓存 key 空间
	b.WriteString(hex.EncodeToString(sum[:8]))
	return b.String()
}

func clearCache(cache *lru.Cache[string, string]) {
	cache.Purge()
}

func getCacheSize(cache *lru.Cache[string, string]) int {
	return cache.Len()
}

func extractWithFileCheck(filePath string, extractor extractFunc) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	content, err := extractor(filePath)
	if err != nil {
		return newErrorResult(ctx, fmt.Sprintf("提取内容失败: %v", err)), err
	}

	return newSuccessResult(ctx, content), nil
}

// resolveMimeType 安全解析文件的 MIME 类型：去除扩展名前缀的 `.`，
// 避免 ext[1:] 在无扩展名时 panic；空扩展名回退为 application/octet-stream。
func resolveMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return "application/octet-stream"
	}
	return MapExtensionToMimeType(strings.TrimPrefix(ext, "."))
}
