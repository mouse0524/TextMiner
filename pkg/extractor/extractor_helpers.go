package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"textminer/pkg/logger"
	"time"
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
		Status:   "success",
		Content:  content,
	}
}

func newErrorResult(ctx *ExtractContext, errMsg string) *ExtractResult {
	return &ExtractResult{
		FileName:     ctx.FileName,
		FileType:     ctx.FileType,
		FileSize:     ctx.FileSize,
		Status:       "failed",
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
		Status:       "failed",
		ErrorMessage: "文件不存在或无法访问",
	}
}

func extractWithCache(cache *sync.Map, filePath string, extractor extractFunc) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := fmt.Sprintf("%s:%d", filePath, fileInfo.ModTime().UnixNano())
	if cached, ok := cache.Load(cacheKey); ok {
		if content, ok := cached.(string); ok {
			logger.Infof("使用缓存内容: %s", filePath)
			return content, nil
		}
	}

	startTime := time.Now()
	content, err := extractor(filePath)
	if err != nil {
		return "", err
	}
	elapsed := time.Since(startTime)
	logger.Infof("提取完成: %s, 耗时: %v, 内容长度: %d", filePath, elapsed, len(content))

	cache.Store(cacheKey, content)
	return content, nil
}

func extractWithCacheAndOcr(cache *sync.Map, filePath string, enableOcr bool, extractor extractFuncWithOcr) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := fmt.Sprintf("%s:%d:%v", filePath, fileInfo.ModTime().UnixNano(), enableOcr)
	if cached, ok := cache.Load(cacheKey); ok {
		if content, ok := cached.(string); ok {
			logger.Infof("使用缓存内容: %s", filePath)
			return content, nil
		}
	}

	startTime := time.Now()
	content, err := extractor(filePath, enableOcr)
	if err != nil {
		return "", err
	}
	elapsed := time.Since(startTime)
	logger.Infof("提取完成: %s, 耗时: %v, 内容长度: %d", filePath, elapsed, len(content))

	cache.Store(cacheKey, content)
	return content, nil
}

func clearCache(cache *sync.Map) {
	cache.Range(func(key, value interface{}) bool {
		cache.Delete(key)
		return true
	})
}

func getCacheSize(cache *sync.Map) int {
	size := 0
	cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
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
