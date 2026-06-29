package extractor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"textminer/pkg/logger"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/errgroup"
)

// DocExtractor DOC文件提取器
type DocExtractor struct{}

// Extract 提取DOC文件内容
func (e *DocExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	return extractWithFileCheck(filePath, extractWordDocContent)
}

// DocxExtractor DOCX文件提取器
type DocxExtractor struct{}

// Extract 提取DOCX文件内容，直接解析XML结构提取文本
func (e *DocxExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	content, err := extractDocxOrOdtContent(filePath, enableOcr)
	if err != nil {
		return newErrorResult(ctx, err.Error()), err
	}

	return newSuccessResult(ctx, content), nil
}

func extractDocxOrOdtContent(filePath string, enableOcr bool) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer reader.Close()

	isOdt := false
	for _, file := range reader.File {
		if file.Name == "content.xml" {
			isOdt = true
			break
		}
	}

	var content strings.Builder
	content.Grow(1024 * 1024)

	if isOdt {
		return extractOdtContent(filePath, reader, enableOcr)
	}
	return extractDocxContent(filePath, reader, enableOcr)
}

func extractOdtContent(filePath string, reader *zip.ReadCloser, enableOcr bool) (string, error) {
	text, err := extractODTContentWithCache(filePath)
	if err != nil {
		return "", fmt.Errorf("解析ODT文件失败: %w", err)
	}

	var content strings.Builder
	content.WriteString(text)

	if enableOcr {
		ocrContent, err := extractOfficeImagesWithOcr(reader, "Pictures")
		if err == nil && ocrContent != "" {
			content.WriteString(ocrContent)
		}
	}

	return content.String(), nil
}

func extractDocxContent(filePath string, reader *zip.ReadCloser, enableOcr bool) (string, error) {
	var content strings.Builder
	content.Grow(1024 * 1024)

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "word/document.xml") {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("读取文档内容失败: %w", err)
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return "", fmt.Errorf("读取文件数据失败: %w", err)
		}

		text := extractTextFromXML(data)
		content.WriteString(text)
		content.WriteString("\n")
	}

	if enableOcr {
		ocrContent, err := extractOfficeImagesWithOcr(reader, "word/media")
		if err == nil && ocrContent != "" {
			content.WriteString(ocrContent)
		}
	}

	embeddingExtractor := NewOfficeEmbeddingExtractor("docx")
	embeddingExtractor.ExtractFromOfficeFile(reader, &content, 0)

	return content.String(), nil
}

// extractTextFromXML 从XML数据中提取文本内容
func extractTextFromXML(data []byte) string {
	var result strings.Builder
	result.Grow(len(data) / 4)

	dataBytes := data
	startTag := []byte("<w:t>")
	endTag := []byte("</w:t>")
	startTagLen := len(startTag)
	endTagLen := len(endTag)

	for {
		start := bytes.Index(dataBytes, startTag)
		if start == -1 {
			break
		}

		end := bytes.Index(dataBytes[start+startTagLen:], endTag)
		if end == -1 {
			break
		}
		actualEnd := start + startTagLen + end

		text := dataBytes[start+startTagLen : actualEnd]
		result.Write(text)

		dataBytes = dataBytes[actualEnd+endTagLen:]
	}

	return result.String()
}

// extractODTTextFast 使用字节操作快速提取ODT文本内容
func extractODTTextFast(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.Grow(len(data) / 4)

	inTag := false
	skipWhitespace := true
	for _, b := range data {
		if b == '<' {
			inTag = true
			skipWhitespace = true
		} else if b == '>' {
			inTag = false
		} else if !inTag {
			if b == '\n' || b == '\r' {
				if !skipWhitespace {
					result.WriteByte(' ')
				}
				skipWhitespace = true
			} else if b == ' ' || b == '\t' {
				if !skipWhitespace {
					result.WriteByte(' ')
					skipWhitespace = true
				}
			} else {
				result.WriteByte(b)
				skipWhitespace = false
			}
		}
	}

	return strings.TrimSpace(result.String()), nil
}

// PptExtractor PPT文件提取器
type PptExtractor struct{}

// Extract 提取PPT文件内容
func (e *PptExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	return extractWithFileCheck(filePath, extractPowerPointPptContentWithCache)
}

// PptxExtractor PPTX文件提取器
type PptxExtractor struct{}

// Extract 提取PPTX文件内容，直接解析XML结构提取文本
func (e *PptxExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	isVisio := ext == ".vsdx"

	logger.Infof("开始提取PPTX文件: %s, 大小: %d bytes", filePath, ctx.FileSize)

	if ctx.FileSize > 100*1024*1024 {
		logger.Warnf("PPTX文件较大: %d bytes，可能需要较长时间", ctx.FileSize)
	}

	var content string
	if isVisio {
		content, err = extractVisioContent(filePath, enableOcr)
	} else {
		content, err = extractPPTXContentWithCache(filePath, enableOcr)
	}

	if err != nil {
		return newErrorResult(ctx, fmt.Sprintf("提取PPTX文件内容失败: %v", err)), err
	}

	logger.Infof("PPTX提取完成: %s, 内容长度: %d", filePath, len(content))
	return newSuccessResult(ctx, content), nil
}

func extractVisioContent(filePath string, enableOcr bool) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer reader.Close()

	var content strings.Builder
	content.Grow(1024 * 1024)

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "visio/pages/page") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		f, err := file.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		text := extractTextFromXML(data)
		content.WriteString(text)
		content.WriteString("\n")
	}

	if enableOcr {
		ocrContent, err := extractOfficeImagesWithOcr(reader, "visio/media")
		if err == nil && ocrContent != "" {
			content.WriteString(ocrContent)
		}
	}

	return strings.TrimSpace(content.String()), nil
}

// extractTextFromPPTXXML 从PPTX XML数据中提取文本内容
func extractTextFromPPTXXML(data []byte) string {
	return extractTextFromXMLTags(data, "<a:t>", "</a:t>")
}

// extractTextFromXMLTags 从XML数据中提取指定标签之间的文本内容
func extractTextFromXMLTags(data []byte, startTag, endTag string) string {
	var result strings.Builder
	result.Grow(len(data) / 4)

	dataBytes := data
	startTagBytes := []byte(startTag)
	endTagBytes := []byte(endTag)
	startTagLen := len(startTagBytes)
	endTagLen := len(endTagBytes)

	for {
		start := bytes.Index(dataBytes, startTagBytes)
		if start == -1 {
			break
		}

		end := bytes.Index(dataBytes[start+startTagLen:], endTagBytes)
		if end == -1 {
			break
		}
		actualEnd := start + startTagLen + end

		text := dataBytes[start+startTagLen : actualEnd]
		result.Write(text)

		dataBytes = dataBytes[actualEnd+endTagLen:]
	}

	return result.String()
}

// XlsExtractor XLS文件提取器
type XlsExtractor struct{}

// Extract 提取XLS文件内容
func (e *XlsExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	content, err := extractExcelXlsContent(filePath)
	if err != nil {
		return newErrorResult(ctx, fmt.Sprintf("提取XLS文件内容失败: %v", err)), err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return newErrorResult(ctx, "未找到Excel内容"), fmt.Errorf("未找到Excel内容")
	}

	return newSuccessResult(ctx, content), nil
}

// XlsxExtractor XLSX文件提取器
type XlsxExtractor struct{}

// Extract 提取XLSX文件内容
func (e *XlsxExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	content, err := extractXlsxContent(filePath, enableOcr)
	if err != nil {
		return newErrorResult(ctx, err.Error()), err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return newErrorResult(ctx, "未找到Excel内容"), fmt.Errorf("未找到Excel内容")
	}

	return newSuccessResult(ctx, content), nil
}

func extractXlsxContent(filePath string, enableOcr bool) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开XLSX失败: %w", err)
	}
	defer reader.Close()

	var content strings.Builder
	content.Grow(1024 * 1024)

	sharedStrings := parseSharedStringsFromZip(reader)

	sheetCount := 0
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".xml") {
			sheetCount++
			text := extractSheetContent(file, sharedStrings)
			if text != "" {
				content.WriteString(text)
				content.WriteString("\n")
			}
		}
	}

	if sheetCount == 0 || content.Len() == 0 {
		extractFallbackXlsxContent(reader, &content)
	}

	if enableOcr {
		ocrContent, err := extractOfficeImagesWithOcr(reader, "xl/media")
		if err == nil && ocrContent != "" {
			content.WriteString(ocrContent)
		}
	}

	embeddingExtractor := NewOfficeEmbeddingExtractor("xlsx")
	embeddingExtractor.ExtractFromOfficeFile(reader, &content, 0)

	return content.String(), nil
}

func parseSharedStringsFromZip(reader *zip.ReadCloser) map[int]string {
	sharedStrings := make(map[int]string)

	for _, file := range reader.File {
		if file.Name == "xl/sharedStrings.xml" {
			f, err := file.Open()
			if err != nil {
				continue
			}

			data, _ := io.ReadAll(f)
			f.Close()
			sharedStrings = parseSharedStrings(data)
			break
		}
	}

	return sharedStrings
}

func extractSheetContent(file *zip.File, sharedStrings map[int]string) string {
	f, err := file.Open()
	if err != nil {
		return ""
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}

	return extractSheetText(data, sharedStrings)
}

func extractFallbackXlsxContent(reader *zip.ReadCloser, content *strings.Builder) {
	for _, file := range reader.File {
		if !strings.Contains(file.Name, "xl/") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		f, err := file.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		text := extractTextFromXMLTags(data, "<v>", "</v>")
		if text != "" {
			content.WriteString(decodeHTMLEntities(text))
			content.WriteString(" ")
		}

		text2 := extractTextFromXMLTags(data, "<t>", "</t>")
		if text2 != "" {
			content.WriteString(decodeHTMLEntities(text2))
			content.WriteString(" ")
		}
	}
}

// parseSharedStrings 解析共享字符串表
func parseSharedStrings(data []byte) map[int]string {
	sharedStrings := make(map[int]string)

	startTag := []byte("<si>")
	endTag := []byte("</si>")
	tStartTag := []byte("<t")
	tEndTag := []byte("</t>")

	index := 0

	dataBytes := data
	for {
		siStart := bytes.Index(dataBytes, startTag)
		if siStart == -1 {
			break
		}

		siEnd := bytes.Index(dataBytes[siStart+len(startTag):], endTag)
		if siEnd == -1 {
			break
		}
		actualSiEnd := siStart + len(startTag) + siEnd

		siContent := dataBytes[siStart+len(startTag) : actualSiEnd]

		// 查找<t>标签内的文本
		tStartPos := bytes.Index(siContent, tStartTag)
		if tStartPos != -1 {
			tEndPos := bytes.Index(siContent[tStartPos:], tEndTag)
			if tEndPos != -1 {
				tStartPos = tStartPos + 2 // 跳过"<t"

				// 处理可能的属性，找到">"的位置
				gtPos := bytes.IndexByte(siContent[tStartPos:], '>')
				if gtPos != -1 {
					tStartPos = tStartPos + gtPos + 1
					// tEndPos是从tStartPos之后开始的，所以需要重新计算
					tEndPos = bytes.Index(siContent[tStartPos:], tEndTag)
					if tEndPos != -1 {
						text := siContent[tStartPos : tStartPos+tEndPos]
						sharedStrings[index] = string(text)
					}
				}
			}
		}

		index++
		dataBytes = dataBytes[actualSiEnd+len(endTag):]
	}

	return sharedStrings
}

// extractSheetText 从工作表XML中提取文本
func extractSheetText(data []byte, sharedStrings map[int]string) string {
	var result strings.Builder

	cellStartTag := []byte("<c ")
	cellEndTag := []byte("</c>")
	tAttrStartTag := []byte("t=\"")

	dataBytes := data
	for {
		cStart := bytes.Index(dataBytes, cellStartTag)
		if cStart == -1 {
			break
		}

		cEnd := bytes.Index(dataBytes[cStart:], cellEndTag)
		if cEnd == -1 {
			break
		}
		actualCEnd := cStart + cEnd + len(cellEndTag)

		cellContent := dataBytes[cStart:actualCEnd]

		tPos := bytes.Index(cellContent, tAttrStartTag)
		if tPos != -1 {
			tPos += len(tAttrStartTag)
			tEnd := bytes.IndexByte(cellContent[tPos:], '"')
			if tEnd != -1 {
				tValue := string(cellContent[tPos : tPos+tEnd])

				if tValue == "s" {
					vStart := bytes.Index(cellContent, []byte("<v>"))
					if vStart != -1 {
						vEnd := bytes.Index(cellContent[vStart+3:], []byte("</v>"))
						if vEnd != -1 {
							vStart += 3
							numStr := string(cellContent[vStart : vStart+vEnd])
							if num, err := strconv.Atoi(numStr); err == nil {
								if str, exists := sharedStrings[num]; exists {
									result.WriteString(str)
									result.WriteString(" ")
								}
							}
						}
					}
				} else if tValue == "str" {
					vStart := bytes.Index(cellContent, []byte("<v>"))
					if vStart != -1 {
						vEnd := bytes.Index(cellContent[vStart+3:], []byte("</v>"))
						if vEnd != -1 {
							vStart += 3
							strValue := string(cellContent[vStart : vStart+vEnd])
							result.WriteString(decodeHTMLEntities(strValue))
							result.WriteString(" ")
						}
					}
				}
			}
		}

		dataBytes = dataBytes[actualCEnd:]
	}

	return result.String()
}

// isPureNumber 检查字符串是否为纯数字
func isPureNumber(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func decodeHTMLEntities(s string) string {
	result := html.UnescapeString(s)

	result = strings.ReplaceAll(result, "\\u0026", "&")

	if !strings.Contains(result, "\\u") {
		return result
	}

	var builder strings.Builder
	builder.Grow(len(result))

	start := 0
	for {
		idx := strings.Index(result[start:], "\\u")
		if idx == -1 {
			builder.WriteString(result[start:])
			break
		}

		idx += start
		builder.WriteString(result[start:idx])

		end := idx + 6
		if end > len(result) {
			builder.WriteString(result[idx:])
			break
		}

		hexStr := result[idx+2 : end]
		code, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			builder.WriteString(result[idx:end])
			start = end
			continue
		}

		builder.WriteRune(rune(code))
		start = end
	}

	return builder.String()
}

func extractOfficeImagesWithOcr(reader *zip.ReadCloser, mediaPath string) (string, error) {
	ocrProcessor, err := GetOcrProcessor()
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "office_ocr")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var content strings.Builder

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, mediaPath) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".bmp" && ext != ".gif" {
			continue
		}

		tempImagePath := filepath.Join(tempDir, filepath.Base(file.Name))

		src, err := file.Open()
		if err != nil {
			continue
		}

		dst, err := os.Create(tempImagePath)
		if err != nil {
			src.Close()
			continue
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			continue
		}

		ocrText, err := ocrProcessor.Recognize(tempImagePath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(ocrText) != "" {
			content.WriteString(ocrText)
			content.WriteString("\n")
		}
	}

	return content.String(), nil
}

// VsdExtractor VSD文件提取器
type VsdExtractor struct{}

// Extract 提取VSD文件内容
func (e *VsdExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	return newErrorResult(ctx, "VSD文件格式不支持（需要安装Microsoft Visio或转换为VSX格式）"),
		fmt.Errorf("VSD文件格式不支持（需要安装Microsoft Visio或转换为VSX格式）")
}

// XlsbExtractor XLSB文件提取器
type XlsbExtractor struct{}

// Extract 提取XLSB文件内容
func (e *XlsbExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}

	content, err := extractExcelXlsbContent(filePath)
	if err != nil {
		return newErrorResult(ctx, fmt.Sprintf("提取XLSB文件内容失败: %v", err)), err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return newErrorResult(ctx, "未找到Excel内容"), fmt.Errorf("未找到Excel内容")
	}

	return newSuccessResult(ctx, content), nil
}

var pptxContentCache *lru.Cache[string, string]

func init() {
	pptxContentCache, _ = lru.New[string, string](128)
}

func extractPPTXContentWithCache(filePath string, enableOcr bool) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := buildCacheKey(filePath, fileInfo.ModTime().UnixNano(), fileInfo.Size(), enableOcr)
	if cached, ok := pptxContentCache.Get(cacheKey); ok {
		logger.Infof("使用缓存的PPTX内容: %s", filePath)
		return cached, nil
	}

	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	estimatedSize := int(fileInfo.Size()) / 10
	if estimatedSize < 1024*1024 {
		estimatedSize = 1024 * 1024
	}
	var content strings.Builder
	content.Grow(estimatedSize)

	var slideFiles []*zip.File
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "ppt/slides/slide") && strings.HasSuffix(file.Name, ".xml") {
			slideFiles = append(slideFiles, file)
		}
	}

	type slideResult struct {
		text string
		err  error
	}

	// 使用 errgroup 限制并发到 NumCPU*2，替换裸 goroutine + resultChan 模式
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(runtime.NumCPU() * 2)
	resultChan := make(chan slideResult, len(slideFiles))

	for _, slideFile := range slideFiles {
		slideFile := slideFile
		eg.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("PPTX slide 处理 panic: %v", r)
				}
			}()

			f, err := slideFile.Open()
			if err != nil {
				resultChan <- slideResult{err: err}
				return nil
			}
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				resultChan <- slideResult{err: err}
				return nil
			}

			text := extractTextFromPPTXXML(data)
			resultChan <- slideResult{text: text}
			return nil
		})
	}

	// 等待所有 goroutine 完成
	go func() {
		_ = eg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err == nil && result.text != "" {
			content.WriteString(decodeHTMLEntities(result.text))
			content.WriteString("\n")
		}
	}

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "ppt/embeddings/") || !strings.HasSuffix(file.Name, ".xlsx") {
			continue
		}

		f, err := file.Open()
		if err != nil {
			continue
		}

		excelData, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		excelReader, err := zip.NewReader(bytes.NewReader(excelData), int64(len(excelData)))
		if err != nil {
			continue
		}

		for _, excelFile := range excelReader.File {
			fileName := excelFile.Name
			if !strings.Contains(fileName, "sharedStrings.xml") &&
				!strings.Contains(fileName, "sheet1.xml") &&
				!strings.Contains(fileName, "sheet2.xml") &&
				!strings.Contains(fileName, "sheet3.xml") {
				continue
			}

			xf, err := excelFile.Open()
			if err != nil {
				continue
			}

			xmlData, err := io.ReadAll(xf)
			xf.Close()
			if err != nil {
				continue
			}

			text := extractTextFromXMLTags(xmlData, "<v>", "</v>")
			if text != "" {
				content.WriteString(decodeHTMLEntities(text))
				content.WriteString(" ")
			}

			text2 := extractTextFromXMLTags(xmlData, "<t>", "</t>")
			if text2 != "" {
				content.WriteString(decodeHTMLEntities(text2))
				content.WriteString(" ")
			}
		}
	}

	if enableOcr {
		ocrContent, err := extractOfficeImagesWithOcr(reader, "ppt/media")
		if err == nil && ocrContent != "" {
			content.WriteString(ocrContent)
		}
	}

	embeddingExtractor := NewOfficeEmbeddingExtractor("pptx")
	embeddingExtractor.ExtractFromOfficeFile(reader, &content, 0)

	finalContent := strings.TrimSpace(content.String())
	pptxContentCache.Add(cacheKey, finalContent)
	return finalContent, nil
}

func ClearPPTXCache() {
	clearCache(pptxContentCache)
}

func GetPPTXCacheSize() int {
	return getCacheSize(pptxContentCache)
}

func ClearAllPPTCache() {
	ClearPPTCache()
	ClearPPTXCache()
}

func GetAllPPTCacheSize() int {
	return GetPPTCacheSize() + GetPPTXCacheSize()
}

var odtContentCache *lru.Cache[string, string]

func init() {
	odtContentCache, _ = lru.New[string, string](128)
}

func extractODTContentWithCache(filePath string) (string, error) {
	return extractWithCache(odtContentCache, filePath, extractODTContent)
}

func extractODTContent(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var content strings.Builder
	content.Grow(1024 * 1024)

	for _, file := range reader.File {
		if file.Name != "content.xml" {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return "", err
		}

		text, err := extractODTTextFast(f)
		f.Close()
		if err != nil {
			return "", err
		}

		content.WriteString(text)
		break
	}

	return strings.TrimSpace(content.String()), nil
}

func ClearODTCache() {
	clearCache(odtContentCache)
}

func GetODTCacheSize() int {
	return getCacheSize(odtContentCache)
}
