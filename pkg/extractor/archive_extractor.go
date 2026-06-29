package extractor

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/cavaliergopher/rpm"
	"github.com/kdomanski/iso9660"
	"github.com/nwaples/rardecode/v2"
	"github.com/ulikunitz/xz"

	"textminer/pkg/logger"
)

const (
	MaxArchiveDepth = 10
)

// ArchiveExtractor 压缩包提取器
type ArchiveExtractor struct {
	archiveType string
}

// ArchiveFile 压缩包内文件信息
type ArchiveFile struct {
	Name     string
	Content  string
	IsDir    bool
	Depth    int
	FilePath string
}

// NewArchiveExtractor 创建压缩包提取器
func NewArchiveExtractor(archiveType string) *ArchiveExtractor {
	return &ArchiveExtractor{
		archiveType: archiveType,
	}
}

// Extract 提取压缩包内容
func (e *ArchiveExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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

	files, err := e.extractArchive(filePath, 0)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("解压失败: %v", err)
		return result, err
	}

	var content strings.Builder

	for _, file := range files {
		if !file.IsDir && file.Content != "" {
			content.WriteString(fmt.Sprintf("=== %s ===\n", file.Name))
			content.WriteString(file.Content)
			content.WriteString("\n\n")
		}
	}

	result.Content = content.String()
	return result, nil
}

// extractArchive 提取压缩包内容（支持递归）
func (e *ArchiveExtractor) extractArchive(filePath string, depth int) ([]ArchiveFile, error) {
	if depth >= MaxArchiveDepth {
		return nil, fmt.Errorf("压缩包层级超过最大深度 %d", MaxArchiveDepth)
	}

	switch e.archiveType {
	case "zip":
		return e.extractZip(filePath, depth)
	case "7z":
		return e.extract7z(filePath, depth)
	case "rar":
		return e.extractRar(filePath, depth)
	case "tar":
		return e.extractTar(filePath, depth)
	case "gz":
		return e.extractGz(filePath, depth)
	case "tar.gz", "tgz":
		return e.extractTarGz(filePath, depth)
	case "bz2":
		return e.extractBz2(filePath, depth)
	case "xz":
		return e.extractXz(filePath, depth)
	case "tar.xz":
		return e.extractTarXz(filePath, depth)
	case "tar.bz2":
		return e.extractTarBz2(filePath, depth)
	case "rpm":
		return e.extractRpm(filePath, depth)
	case "iso":
		return e.extractIso(filePath, depth)
	default:
		return nil, fmt.Errorf("不支持的压缩包格式: %s", e.archiveType)
	}
}

// extractZip 提取ZIP文件
func (e *ArchiveExtractor) extractZip(filePath string, depth int) ([]ArchiveFile, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开ZIP文件失败: %w", err)
	}
	defer reader.Close()

	// Zip Bomb 防护：单层文件数限制
	if err := CheckArchiveFileCount(len(reader.File)); err != nil {
		return nil, err
	}

	var files []ArchiveFile
	var totalUncompressed int64

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			files = append(files, ArchiveFile{
				Name:     file.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: file.Name,
			})
			continue
		}

		// SafeReadZipEntry 内置 Zip Slip 校验与 Zip Bomb 防护
		safeName, content, err := SafeReadZipEntry(file, &totalUncompressed)
		if err != nil {
			logger.Warnf("跳过压缩包内文件 %s: %v", file.Name, err)
			continue
		}
		_ = safeName

		archiveFile := ArchiveFile{
			Name:     file.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: file.Name,
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, file.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extract7z 提取7z文件
func (e *ArchiveExtractor) extract7z(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开7z文件失败: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	reader, err := sevenzip.NewReader(file, fileInfo.Size())
	if err != nil {
		return nil, fmt.Errorf("创建7z读取器失败: %w", err)
	}

	var files []ArchiveFile
	var totalUncompressed int64

	for _, file := range reader.File {
		if file.Mode().IsDir() {
			files = append(files, ArchiveFile{
				Name:     file.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: file.Name,
			})
			continue
		}

		rc, err := file.Open()
		if err != nil {
			continue
		}

		content, err := SafeReadLimited(rc, &totalUncompressed)
		rc.Close()
		if err != nil {
			logger.Warnf("跳过7z内文件 %s: %v", file.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     file.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: file.Name,
		}

		ext := strings.ToLower(filepath.Ext(file.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, file.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extractRar 提取RAR文件
func (e *ArchiveExtractor) extractRar(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开RAR文件失败: %w", err)
	}
	defer file.Close()

	reader, err := rardecode.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建RAR读取器失败: %w", err)
	}

	var files []ArchiveFile
	var totalUncompressed int64

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
				return nil, ErrEncrypted
			}
			continue
		}

		if header.IsDir {
			files = append(files, ArchiveFile{
				Name:     header.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: header.Name,
			})
			continue
		}

		content, err := SafeReadLimited(reader, &totalUncompressed)
		if err != nil {
			if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
				return nil, ErrEncrypted
			}
			logger.Warnf("跳过RAR内文件 %s: %v", header.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     header.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: header.Name,
		}

		ext := strings.ToLower(filepath.Ext(header.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, header.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extractTar 提取TAR文件
func (e *ArchiveExtractor) extractTar(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TAR文件失败: %w", err)
	}
	defer file.Close()

	reader := tar.NewReader(file)

	var files []ArchiveFile
	var totalUncompressed int64

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if header.Typeflag == tar.TypeDir {
			files = append(files, ArchiveFile{
				Name:     header.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: header.Name,
			})
			continue
		}

		content, err := SafeReadLimited(reader, &totalUncompressed)
		if err != nil {
			logger.Warnf("跳过TAR内文件 %s: %v", header.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     header.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: header.Name,
		}

		ext := strings.ToLower(filepath.Ext(header.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, header.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extractGz 提取GZ文件
func (e *ArchiveExtractor) extractGz(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开GZ文件失败: %w", err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建GZ读取器失败: %w", err)
	}
	defer reader.Close()

	var totalUncompressed int64
	content, err := SafeReadLimited(reader, &totalUncompressed)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, ".gz")
	baseName = strings.TrimSuffix(baseName, ".GZ")

	fileContent := e.extractFileContent(content, baseName)

	if fileContent == "" {
		fileContent = fmt.Sprintf("[GZ文件已解压，大小: %d 字节]", len(content))
	}

	return []ArchiveFile{
		{
			Name:     baseName,
			Content:  fileContent,
			IsDir:    false,
			Depth:    depth,
			FilePath: baseName,
		},
	}, nil
}

// extractTarGz 提取TAR.GZ文件
func (e *ArchiveExtractor) extractTarGz(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TAR.GZ文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建GZ读取器失败: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var files []ArchiveFile
	var totalUncompressed int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if header.Typeflag == tar.TypeDir {
			files = append(files, ArchiveFile{
				Name:     header.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: header.Name,
			})
			continue
		}

		content, err := SafeReadLimited(tarReader, &totalUncompressed)
		if err != nil {
			logger.Warnf("跳过TAR.GZ内文件 %s: %v", header.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     header.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: header.Name,
		}

		ext := strings.ToLower(filepath.Ext(header.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, header.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extractNestedArchive 提取嵌套压缩包
func (e *ArchiveExtractor) extractNestedArchive(content []byte, ext string, depth int) ([]ArchiveFile, error) {
	tmpFile, err := os.CreateTemp("", "archive-*"+ext)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(content)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}
	tmpFile.Close()

	var archiveType string
	switch ext {
	case ".zip":
		archiveType = "zip"
	case ".7z":
		archiveType = "7z"
	case ".rar":
		archiveType = "rar"
	case ".tar":
		archiveType = "tar"
	case ".gz":
		archiveType = "gz"
	case ".tgz", ".tar.gz":
		archiveType = "tar.gz"
	case ".bz2":
		archiveType = "bz2"
	case ".xz":
		archiveType = "xz"
	case ".tar.xz":
		archiveType = "tar.xz"
	case ".tar.bz2":
		archiveType = "tar.bz2"
	case ".rpm":
		archiveType = "rpm"
	case ".iso":
		archiveType = "iso"
	default:
		return nil, fmt.Errorf("不支持的嵌套压缩包格式: %s", ext)
	}

	nestedExtractor := &ArchiveExtractor{archiveType: archiveType}
	return nestedExtractor.extractArchive(tmpFile.Name(), depth)
}

// extractFileContent 根据文件类型提取文件内容
func (e *ArchiveExtractor) extractFileContent(content []byte, fileName string) string {
	fileNameLower := strings.ToLower(fileName)

	doubleExts := []string{".tar.gz", ".tar.xz", ".tar.bz2", ".tgz"}
	for _, doubleExt := range doubleExts {
		if strings.HasSuffix(fileNameLower, doubleExt) {
			ext := strings.TrimPrefix(doubleExt, ".")
			if ext == "tgz" {
				ext = "tar.gz"
			}
			extractor, err := NewExtractorByType(ext)
			if err != nil {
				return fmt.Sprintf("[文件: %s, 大小: %d 字节]", fileName, len(content))
			}

			tmpFile, err := os.CreateTemp("", "extract-*"+ext)
			if err != nil {
				return fmt.Sprintf("[文件: %s, 大小: %d 字节]", fileName, len(content))
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(content)
			if err != nil {
				tmpFile.Close()
				return fmt.Sprintf("[文件: %s, 大小: %d 字节]", fileName, len(content))
			}
			tmpFile.Close()

			result, err := extractor.Extract(tmpFile.Name(), false)
			if err != nil {
				return fmt.Sprintf("[文件: %s, 大小: %d 字节]", fileName, len(content))
			}

			if result.Content != "" {
				return result.Content
			}

			return fmt.Sprintf("[文件: %s, 大小: %d 字节]", fileName, len(content))
		}
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		// 对于无扩展名的文件，尝试检测文件类型
		fileType := e.detectFileType(content, fileName)
		if fileType != "" {
			extractor, err := NewExtractorByType(fileType)
			if err == nil {
				tmpFile, err := os.CreateTemp("", "extract-*")
				if err == nil {
					defer os.Remove(tmpFile.Name())
					_, err = tmpFile.Write(content)
					if err == nil {
						tmpFile.Close()
						result, err := extractor.Extract(tmpFile.Name(), false)
						if err == nil && result.Content != "" {
							return result.Content
						}
					}
				}
			}
		}
		return e.tryExtractTextContent(content, fileName)
	}

	if ext == "" {
		return e.tryExtractTextContent(content, fileName)
	}
	ext = strings.TrimPrefix(ext, ".")

	extractor, err := NewExtractorByType(ext)
	if err != nil {
		return e.tryExtractTextContent(content, fileName)
	}

	tmpFile, err := os.CreateTemp("", "extract-*"+ext)
	if err != nil {
		return e.tryExtractTextContent(content, fileName)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(content)
	if err != nil {
		tmpFile.Close()
		return e.tryExtractTextContent(content, fileName)
	}
	tmpFile.Close()

	result, err := extractor.Extract(tmpFile.Name(), false)
	if err != nil {
		return e.tryExtractTextContent(content, fileName)
	}

	if result.Content != "" {
		return result.Content
	}

	return e.tryExtractTextContent(content, fileName)
}

func (e *ArchiveExtractor) tryExtractTextContent(content []byte, fileName string) string {
	if len(content) == 0 {
		return fmt.Sprintf("[文件: %s, 大小: 0 字节]", fileName)
	}

	str := string(content)

	if e.isPrintableASCII(str) {
		return str
	}

	return fmt.Sprintf("[文件: %s, 大小: %d 字节 (二进制或非文本文件)]", fileName, len(content))
}

func (e *ArchiveExtractor) isPrintableASCII(s string) bool {
	if len(s) == 0 {
		return false
	}

	// 限定取样到前 1000 字节，避免长字符串扫描；同时让分母与分子保持一致
	limit := len(s)
	if limit > 1000 {
		limit = 1000
	}
	sample := s[:limit]

	printableCount := 0
	for i := 0; i < len(sample); i++ {
		c := sample[i]
		if (c >= 32 && c <= 126) || c == '\n' || c == '\r' || c == '\t' {
			printableCount++
		}
	}

	return printableCount*10 >= len(sample)*8
}

// detectFileType 根据文件内容检测文件类型
func (e *ArchiveExtractor) detectFileType(content []byte, fileName string) string {
	if len(content) < 10 {
		return ""
	}

	// 检测tar文件
	if len(content) >= 262 && string(content[257:262]) == "ustar" {
		return "tar"
	}

	// 检测ZIP文件
	if len(content) >= 4 && content[0] == 'P' && content[1] == 'K' && content[2] == 0x03 && content[3] == 0x04 {
		return "zip"
	}

	// 检测PDF文件
	if len(content) >= 4 && content[0] == '%' && content[1] == 'P' && content[2] == 'D' && content[3] == 'F' {
		return "pdf"
	}

	// 检测DOC文件
	if len(content) >= 8 && content[0] == 0xD0 && content[1] == 0xCF && content[2] == 0x11 && content[3] == 0xE0 && content[4] == 0xA1 && content[5] == 0xB1 && content[6] == 0x1A && content[7] == 0xE1 {
		return "doc"
	}

	// 检测DOCX文件
	if len(content) >= 4 && content[0] == 'P' && content[1] == 'K' && content[2] == 0x03 && content[3] == 0x04 {
		return "docx"
	}

	// 检测文本文件
	if e.isPrintableASCII(string(content)) {
		return "txt"
	}

	return ""
}

// ZipExtractor ZIP提取器
type ZipExtractor struct {
	*ArchiveExtractor
}

// NewZipExtractor 创建ZIP提取器
func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{
		ArchiveExtractor: NewArchiveExtractor("zip"),
	}
}

// SevenZipExtractor 7z提取器
type SevenZipExtractor struct {
	*ArchiveExtractor
}

// NewSevenZipExtractor 创建7z提取器
func NewSevenZipExtractor() *SevenZipExtractor {
	return &SevenZipExtractor{
		ArchiveExtractor: NewArchiveExtractor("7z"),
	}
}

// RarExtractor RAR提取器
type RarExtractor struct {
	*ArchiveExtractor
}

// NewRarExtractor 创建RAR提取器
func NewRarExtractor() *RarExtractor {
	return &RarExtractor{
		ArchiveExtractor: NewArchiveExtractor("rar"),
	}
}

// TarExtractor TAR提取器
type TarExtractor struct {
	*ArchiveExtractor
}

// NewTarExtractor 创建TAR提取器
func NewTarExtractor() *TarExtractor {
	return &TarExtractor{
		ArchiveExtractor: NewArchiveExtractor("tar"),
	}
}

// GzExtractor GZ提取器
type GzExtractor struct {
	*ArchiveExtractor
}

// NewGzExtractor 创建GZ提取器
func NewGzExtractor() *GzExtractor {
	return &GzExtractor{
		ArchiveExtractor: NewArchiveExtractor("gz"),
	}
}

// TarGzExtractor TAR.GZ提取器
type TarGzExtractor struct {
	*ArchiveExtractor
}

// NewTarGzExtractor 创建TAR.GZ提取器
func NewTarGzExtractor() *TarGzExtractor {
	return &TarGzExtractor{
		ArchiveExtractor: NewArchiveExtractor("tar.gz"),
	}
}

// extractBz2 提取BZ2文件
func (e *ArchiveExtractor) extractBz2(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开BZ2文件失败: %w", err)
	}
	defer file.Close()

	reader := bzip2.NewReader(file)

	var totalUncompressed int64
	content, err := SafeReadLimited(reader, &totalUncompressed)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, ".bz2")
	baseName = strings.TrimSuffix(baseName, ".BZ2")

	fileContent := e.extractFileContent(content, baseName)

	if fileContent == "" {
		fileContent = fmt.Sprintf("[BZ2文件已解压，大小: %d 字节]", len(content))
	}

	return []ArchiveFile{
		{
			Name:     baseName,
			Content:  fileContent,
			IsDir:    false,
			Depth:    depth,
			FilePath: baseName,
		},
	}, nil
}

// extractXz 提取XZ文件
func (e *ArchiveExtractor) extractXz(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开XZ文件失败: %w", err)
	}
	defer file.Close()

	reader, err := xz.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建XZ读取器失败: %w", err)
	}

	var totalUncompressed int64
	content, err := SafeReadLimited(reader, &totalUncompressed)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, ".xz")
	baseName = strings.TrimSuffix(baseName, ".XZ")

	fileContent := e.extractFileContent(content, baseName)

	if fileContent == "" {
		fileContent = fmt.Sprintf("[XZ文件已解压，大小: %d 字节]", len(content))
	}

	return []ArchiveFile{
		{
			Name:     baseName,
			Content:  fileContent,
			IsDir:    false,
			Depth:    depth,
			FilePath: baseName,
		},
	}, nil
}

func (e *ArchiveExtractor) extractTarXz(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TAR.XZ文件失败: %w", err)
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建XZ读取器失败: %w", err)
	}

	tarReader := tar.NewReader(xzReader)

	var files []ArchiveFile
	var totalUncompressed int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if header.Typeflag == tar.TypeDir {
			files = append(files, ArchiveFile{
				Name:     header.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: header.Name,
			})
			continue
		}

		content, err := SafeReadLimited(tarReader, &totalUncompressed)
		if err != nil {
			logger.Warnf("跳过TAR.XZ内文件 %s: %v", header.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     header.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: header.Name,
		}

		ext := strings.ToLower(filepath.Ext(header.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, header.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// extractRpm 提取RPM文件
func (e *ArchiveExtractor) extractRpm(filePath string, depth int) ([]ArchiveFile, error) {
	pkg, err := rpm.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开RPM文件失败: %w", err)
	}

	var info strings.Builder
	info.WriteString(fmt.Sprintf("[RPM包信息]\n"))
	info.WriteString(fmt.Sprintf("名称: %s\n", pkg.Name()))
	info.WriteString(fmt.Sprintf("版本: %s-%s\n", pkg.Version(), pkg.Release()))
	info.WriteString(fmt.Sprintf("架构: %s\n", pkg.Architecture()))
	info.WriteString(fmt.Sprintf("摘要: %s\n", pkg.Summary()))

	if pkg.Description() != "" {
		info.WriteString(fmt.Sprintf("描述: %s\n", pkg.Description()))
	}

	if pkg.License() != "" {
		info.WriteString(fmt.Sprintf("许可证: %s\n", pkg.License()))
	}

	if pkg.Vendor() != "" {
		info.WriteString(fmt.Sprintf("厂商: %s\n", pkg.Vendor()))
	}

	if pkg.URL() != "" {
		info.WriteString(fmt.Sprintf("URL: %s\n", pkg.URL()))
	}

	info.WriteString(fmt.Sprintf("构建时间: %s\n", pkg.BuildTime()))
	info.WriteString(fmt.Sprintf("大小: %d 字节\n", pkg.Size()))

	if len(pkg.Provides()) > 0 {
		info.WriteString(fmt.Sprintf("\n提供:\n"))
		for _, provide := range pkg.Provides() {
			info.WriteString(fmt.Sprintf("  - %s\n", provide))
		}
	}

	if len(pkg.Requires()) > 0 {
		info.WriteString(fmt.Sprintf("\n依赖:\n"))
		for _, req := range pkg.Requires() {
			info.WriteString(fmt.Sprintf("  - %s\n", req))
		}
	}

	return []ArchiveFile{
		{
			Name:     filepath.Base(filePath),
			Content:  info.String(),
			IsDir:    false,
			Depth:    depth,
			FilePath: filepath.Base(filePath),
		},
	}, nil
}

// extractIso 提取ISO文件
func (e *ArchiveExtractor) extractIso(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开ISO文件失败: %w", err)
	}
	defer file.Close()

	image, err := iso9660.OpenImage(file)
	if err != nil {
		// ISO文件格式不正确，返回基本信息
		var info strings.Builder
		info.WriteString(fmt.Sprintf("[ISO镜像信息]\n"))
		info.WriteString(fmt.Sprintf("警告: ISO文件格式不正确: %v\n", err))
		info.WriteString(fmt.Sprintf("文件大小: %d 字节\n", e.getFileSize(filePath)))

		return []ArchiveFile{
			{
				Name:     filepath.Base(filePath),
				Content:  info.String(),
				IsDir:    false,
				Depth:    depth,
				FilePath: filepath.Base(filePath),
			},
		}, nil
	}

	var info strings.Builder
	info.WriteString(fmt.Sprintf("[ISO镜像信息]\n"))

	label, err := image.Label()
	if err == nil {
		info.WriteString(fmt.Sprintf("卷标识: %s\n", label))
	}

	var files []ArchiveFile

	rootDir, err := image.RootDir()
	if err == nil && rootDir != nil {
		err = e.extractIsoDirectory(rootDir, &files, depth, "")
		if err != nil {
			info.WriteString(fmt.Sprintf("\n警告: 提取文件列表失败: %v\n", err))
		}
	}

	if len(files) == 0 {
		return []ArchiveFile{
			{
				Name:     filepath.Base(filePath),
				Content:  info.String(),
				IsDir:    false,
				Depth:    depth,
				FilePath: filepath.Base(filePath),
			},
		}, nil
	}

	for i := range files {
		files[i].Content = info.String() + "\n\n" + files[i].Content
	}

	return files, nil
}

// getFileSize 获取文件大小
func (e *ArchiveExtractor) getFileSize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

func (e *ArchiveExtractor) extractIsoDirectory(dir *iso9660.File, files *[]ArchiveFile, depth int, path string) error {
	children, err := dir.GetChildren()
	if err != nil {
		return err
	}

	var totalUncompressed int64

	for _, child := range children {
		fullPath := filepath.Join(path, child.Name())

		if child.IsDir() {
			*files = append(*files, ArchiveFile{
				Name:     fullPath,
				IsDir:    true,
				Depth:    depth,
				FilePath: fullPath,
			})

			err = e.extractIsoDirectory(child, files, depth, fullPath)
			if err != nil {
				continue
			}
		} else {
			// 读取文件内容（带 Zip Bomb 防护）
			reader := child.Reader()
			contentBytes, err := SafeReadLimited(reader, &totalUncompressed)
			var content string

			if err != nil {
				content = fmt.Sprintf("[文件: %s, 大小: %d 字节, 读取失败: %v]", child.Name(), child.Size(), err)
			} else {
				content = e.extractFileContent(contentBytes, child.Name())
			}

			*files = append(*files, ArchiveFile{
				Name:     fullPath,
				Content:  content,
				IsDir:    false,
				Depth:    depth,
				FilePath: fullPath,
			})
		}
	}

	return nil
}

func (e *ArchiveExtractor) extractTarBz2(filePath string, depth int) ([]ArchiveFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开TAR.BZ2文件失败: %w", err)
	}
	defer file.Close()

	bz2Reader := bzip2.NewReader(file)

	tarReader := tar.NewReader(bz2Reader)

	var files []ArchiveFile
	var totalUncompressed int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if header.Typeflag == tar.TypeDir {
			files = append(files, ArchiveFile{
				Name:     header.Name,
				IsDir:    true,
				Depth:    depth,
				FilePath: header.Name,
			})
			continue
		}

		content, err := SafeReadLimited(tarReader, &totalUncompressed)
		if err != nil {
			logger.Warnf("跳过TAR.BZ2内文件 %s: %v", header.Name, err)
			continue
		}

		archiveFile := ArchiveFile{
			Name:     header.Name,
			Content:  "",
			IsDir:    false,
			Depth:    depth,
			FilePath: header.Name,
		}

		ext := strings.ToLower(filepath.Ext(header.Name))
		if e.isArchiveFile(ext) {
			nestedFiles, err := e.extractNestedArchive(content, ext, depth+1)
			if err == nil {
				archiveFile.Content = fmt.Sprintf("[嵌套压缩包: %d 个文件]", len(nestedFiles))
				files = append(files, archiveFile)
				files = append(files, nestedFiles...)
				continue
			}
		}

		archiveFile.Content = e.extractFileContent(content, header.Name)

		files = append(files, archiveFile)
	}

	return files, nil
}

// isArchiveFile 判断是否为压缩包文件
func (e *ArchiveExtractor) isArchiveFile(ext string) bool {
	archiveExts := []string{".zip", ".7z", ".rar", ".tar", ".gz", ".tgz", ".tar.gz", ".bz2", ".xz", ".tar.xz", ".tar.bz2", ".rpm", ".iso"}
	for _, archiveExt := range archiveExts {
		if ext == archiveExt {
			return true
		}
	}
	return false
}

// Bz2Extractor BZ2提取器
type Bz2Extractor struct {
	*ArchiveExtractor
}

// NewBz2Extractor 创建BZ2提取器
func NewBz2Extractor() *Bz2Extractor {
	return &Bz2Extractor{
		ArchiveExtractor: NewArchiveExtractor("bz2"),
	}
}

// XzExtractor XZ提取器
type XzExtractor struct {
	*ArchiveExtractor
}

// NewXzExtractor 创建XZ提取器
func NewXzExtractor() *XzExtractor {
	return &XzExtractor{
		ArchiveExtractor: NewArchiveExtractor("xz"),
	}
}

// TarBz2Extractor TAR.BZ2提取器
type TarBz2Extractor struct {
	*ArchiveExtractor
}

// NewTarBz2Extractor 创建TAR.BZ2提取器
func NewTarBz2Extractor() *TarBz2Extractor {
	return &TarBz2Extractor{
		ArchiveExtractor: NewArchiveExtractor("tar.bz2"),
	}
}

// TarXzExtractor TAR.XZ提取器
type TarXzExtractor struct {
	*ArchiveExtractor
}

// NewTarXzExtractor 创建TAR.XZ提取器
func NewTarXzExtractor() *TarXzExtractor {
	return &TarXzExtractor{
		ArchiveExtractor: NewArchiveExtractor("tar.xz"),
	}
}

// RpmExtractor RPM提取器
type RpmExtractor struct {
	*ArchiveExtractor
}

// NewRpmExtractor 创建RPM提取器
func NewRpmExtractor() *RpmExtractor {
	return &RpmExtractor{
		ArchiveExtractor: NewArchiveExtractor("rpm"),
	}
}

// IsoExtractor ISO提取器
type IsoExtractor struct {
	*ArchiveExtractor
}

// NewIsoExtractor 创建ISO提取器
func NewIsoExtractor() *IsoExtractor {
	return &IsoExtractor{
		ArchiveExtractor: NewArchiveExtractor("iso"),
	}
}
