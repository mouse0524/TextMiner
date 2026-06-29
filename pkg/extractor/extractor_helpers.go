package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"textminer/pkg/logger"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type extractFunc func(filePath string) (string, error)
type extractFuncWithOcr func(filePath string, enableOcr bool) (string, error)

type ExtractContext struct {
	FileName string
	FileType string
	FileSize int64
	FilePath string
	Ext      string
	MimeType string
}

func prepareExtractContext(filePath string) (*ExtractContext, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	detector := GetFileTypeDetector()
	_, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil || mimeType == "" {
		mimeType = MapExtensionToMimeType(strings.TrimPrefix(ext, "."))
	}

	return &ExtractContext{
		FileName: filepath.Base(filePath),
		FileType: mimeType,
		FileSize: fileInfo.Size(),
		FilePath: filePath,
		Ext:      ext,
		MimeType: mimeType,
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

	cache.Add(cacheKey, content)
	return content, nil
}

func extractWithCacheAndOcr(cache *lru.Cache[string, string], filePath string, enableOcr bool, extractor extractFuncWithOcr) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := buildCacheKey(filePath, fileInfo.ModTime().UnixNano(), fileInfo.Size(), enableOcr)
	if cached, ok := cache.Get(cacheKey); ok {
		logger.Infof("使用缓存内容: %s", filePath)
		return cached, nil
	}

	startTime := time.Now()
	content, err := extractor(filePath, enableOcr)
	if err != nil {
		return "", err
	}
	elapsed := time.Since(startTime)
	logger.Infof("提取完成: %s, 耗时: %v, 内容长度: %d", filePath, elapsed, len(content))

	cache.Add(cacheKey, content)
	return content, nil
}

// buildCacheKey 高效构造 cache key：path + mtime + size + ocr flag。
// 使用 strings.Builder 避免 fmt.Sprintf 的反射开销。
func buildCacheKey(path string, mtimeNano, size int64, ocr bool) string {
	var b strings.Builder
	b.Grow(len(path) + 32)
	b.WriteString(path)
	b.WriteByte(':')
	b.WriteString(strconv.FormatInt(mtimeNano, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatInt(size, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatBool(ocr))
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

func isFileAccessible(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
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
