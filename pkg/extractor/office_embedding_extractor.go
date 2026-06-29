package extractor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"textminer/pkg/logger"
)

// OfficeEmbeddingExtractor Office内嵌附件提取器
type OfficeEmbeddingExtractor struct {
	fileType string
}

// NewOfficeEmbeddingExtractor 创建Office内嵌附件提取器
func NewOfficeEmbeddingExtractor(fileType string) *OfficeEmbeddingExtractor {
	return &OfficeEmbeddingExtractor{
		fileType: fileType,
	}
}

// ExtractFromOfficeFile 从Office文件中提取内嵌附件内容
// depth: 当前递归深度；首次调用传 0，达到 MaxEmbedDepth 时静默跳过。
func (e *OfficeEmbeddingExtractor) ExtractFromOfficeFile(reader *zip.ReadCloser, content *strings.Builder, depth int) error {
	if depth >= MaxEmbedDepth {
		logger.Warnf("Office 嵌入递归达到最大深度 %d，跳过剩余嵌套", MaxEmbedDepth)
		return nil
	}

	var embeddingsPath string

	switch e.fileType {
	case "docx":
		embeddingsPath = "word/embeddings/"
	case "pptx":
		embeddingsPath = "ppt/embeddings/"
	case "xlsx":
		embeddingsPath = "xl/embeddings/"
	default:
		return nil
	}

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, embeddingsPath) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext == ".bin" {
			if err := e.extractFromBinFile(file, content, depth); err != nil {
				continue
			}
		} else if ext == ".xlsx" || ext == ".docx" || ext == ".pptx" {
			if err := e.extractFromOfficeEmbedding(file, content, depth); err != nil {
				continue
			}
		}
	}

	return nil
}

// extractFromBinFile 从.bin文件中提取内容
func (e *OfficeEmbeddingExtractor) extractFromBinFile(binFile *zip.File, content *strings.Builder, depth int) error {
	f, err := binFile.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	binData, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// 检查文件类型
	if len(binData) > 4 {
		// 检查是否是ZIP文件（新格式Office）
		if bytes.HasPrefix(binData, []byte{0x50, 0x4B, 0x03, 0x04}) ||
			bytes.HasPrefix(binData, []byte{0x50, 0x4B, 0x05, 0x06}) ||
			bytes.HasPrefix(binData, []byte{0x50, 0x4B, 0x07, 0x08}) {
			return e.extractFromZipBin(binData, binFile.Name, content, depth)
		}

		// 检查是否是OLE2复合文档（旧格式Office）
		if bytes.HasPrefix(binData, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
			return e.extractFromOleBin(binData, binFile.Name, content)
		}

		// 检查是否是PDF文件
		if bytes.HasPrefix(binData, []byte{0x25, 0x50, 0x44, 0x46}) {
			return e.extractFromPdfBin(binData, binFile.Name, content)
		}
	}

	return nil
}

// extractFromZipBin 从ZIP格式的bin文件中提取内容
func (e *OfficeEmbeddingExtractor) extractFromZipBin(binData []byte, binFileName string, content *strings.Builder, depth int) error {
	binReader, err := zip.NewReader(bytes.NewReader(binData), int64(len(binData)))
	if err != nil {
		return err
	}

	hasOfficeStructure := false
	officeType := ""

	for _, innerFile := range binReader.File {
		if strings.Contains(innerFile.Name, "xl/") {
			hasOfficeStructure = true
			officeType = "xlsx"
			break
		}
		if strings.Contains(innerFile.Name, "word/") {
			hasOfficeStructure = true
			officeType = "docx"
			break
		}
		if strings.Contains(innerFile.Name, "ppt/") {
			hasOfficeStructure = true
			officeType = "pptx"
			break
		}
	}

	if hasOfficeStructure && officeType != "" {
		tmpFile, err := os.CreateTemp("", "office-bin-*."+officeType)
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write(binData)
		if err != nil {
			tmpFile.Close()
			return err
		}
		tmpFile.Close()

		extractor, err := NewExtractorByType(officeType)
		if err != nil {
			return err
		}

		result, err := extractor.Extract(tmpFile.Name(), false)
		if err != nil {
			return err
		}

		if result.Content != "" {
			content.WriteString(result.Content)
			content.WriteString("\n")
		}
	} else {
		for _, innerFile := range binReader.File {
			if innerFile.FileInfo().IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(innerFile.Name))
			if ext == ".bin" || ext == ".package" || ext == ".xml" {
				if err := e.extractPackageContent(innerFile, content, binFileName, depth); err != nil {
					continue
				}
			}
		}
	}

	return nil
}

// extractFromOleBin 从OLE2复合文档中提取内容
func (e *OfficeEmbeddingExtractor) extractFromOleBin(binData []byte, binFileName string, content *strings.Builder) error {
	// 尝试从OLE2复合文档中提取PDF
	// OLE2复合文档的结构比较复杂，这里使用简化的方法
	// 查找PDF文件头标记
	pdfStart := bytes.Index(binData, []byte{0x25, 0x50, 0x44, 0x46})
	if pdfStart == -1 {
		return fmt.Errorf("未在OLE2复合文档中找到PDF文件")
	}

	// 查找PDF结束标记 %%EOF
	pdfEnd := bytes.Index(binData[pdfStart:], []byte{0x25, 0x25, 0x45, 0x4F, 0x46})
	if pdfEnd == -1 {
		// 如果没有找到结束标记，取从PDF开始到文件末尾的所有数据
		pdfData := binData[pdfStart:]
		return e.extractPdfData(pdfData, content)
	}

	// 找到结束标记，取到结束标记之后
	pdfData := binData[pdfStart : pdfStart+pdfEnd+5]
	return e.extractPdfData(pdfData, content)
}

// extractPdfData 从PDF数据中提取内容
func (e *OfficeEmbeddingExtractor) extractPdfData(pdfData []byte, content *strings.Builder) error {
	tmpFile, err := os.CreateTemp("", "ole-pdf-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(pdfData)
	if err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	extractor, err := NewExtractorByType("pdf")
	if err != nil {
		return err
	}

	result, err := extractor.Extract(tmpFile.Name(), false)
	if err != nil {
		return err
	}

	if result.Content != "" {
		content.WriteString(result.Content)
		content.WriteString("\n")
	}

	return nil
}

// extractFromPdfBin 从PDF格式的bin文件中提取内容
func (e *OfficeEmbeddingExtractor) extractFromPdfBin(binData []byte, binFileName string, content *strings.Builder) error {
	return e.extractPdfData(binData, content)
}

// extractFromOfficeEmbedding 从Office内嵌文件中提取内容
func (e *OfficeEmbeddingExtractor) extractFromOfficeEmbedding(embeddingFile *zip.File, content *strings.Builder, depth int) error {
	f, err := embeddingFile.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	embeddingData, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "office-embedding-*"+filepath.Ext(embeddingFile.Name))
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(embeddingData)
	if err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(embeddingFile.Name)), ".")
	extractor, err := NewExtractorByType(ext)
	if err != nil {
		return err
	}

	result, err := extractor.Extract(tmpFile.Name(), false)
	if err != nil {
		return err
	}

	if result.Content != "" {
		content.WriteString(result.Content)
		content.WriteString("\n")
	}

	return nil
}

// extractPackageContent 从package文件中提取内容
func (e *OfficeEmbeddingExtractor) extractPackageContent(packageFile *zip.File, content *strings.Builder, sourceName string, depth int) error {
	f, err := packageFile.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	packageData, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "office-package-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(packageData)
	if err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	detector := &FileTypeDetector{}
	detectedType, err := detector.DetectFileType(tmpFile.Name())
	if err != nil {
		return err
	}

	if detectedType == "unknown" {
		ext := strings.ToLower(filepath.Ext(packageFile.Name))
		if ext != "" {
			detectedType = strings.TrimPrefix(ext, ".")
		}
	}

	extractor, err := NewExtractorByType(detectedType)
	if err != nil {
		return err
	}

	result, err := extractor.Extract(tmpFile.Name(), false)
	if err != nil {
		return err
	}

	if result.Content != "" {
		content.WriteString(result.Content)
		content.WriteString("\n")
	}

	return nil
}
