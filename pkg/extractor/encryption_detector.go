package extractor

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

type EncryptionFeature struct {
	Name       string
	MagicBytes []byte
	FileTypes  []string
}

type EncryptionFeatureLibrary struct {
	features []EncryptionFeature
}

func NewEncryptionFeatureLibrary() *EncryptionFeatureLibrary {
	lib := &EncryptionFeatureLibrary{
		features: make([]EncryptionFeature, 0),
	}
	lib.initDefaultFeatures()
	return lib
}

func (lib *EncryptionFeatureLibrary) initDefaultFeatures() {
	lib.features = []EncryptionFeature{
		{
			Name:       "OLE Compound Document",
			MagicBytes: []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
			FileTypes:  []string{"doc", "xls", "ppt", "docx", "xlsx", "pptx", "docm", "xlsm", "pptm", "xlsb", "wps", "et", "dps"},
		},
		{
			Name:       "ZIP Archive",
			MagicBytes: []byte{0x50, 0x4B, 0x03, 0x04},
			FileTypes:  []string{"zip", "docx", "xlsx", "pptx", "odt", "ods", "odp"},
		},
		{
			Name:       "RAR5 Archive",
			MagicBytes: []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00},
			FileTypes:  []string{"rar"},
		},
		{
			Name:       "RAR4 Archive",
			MagicBytes: []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00},
			FileTypes:  []string{"rar"},
		},
		{
			Name:       "7Z Archive",
			MagicBytes: []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
			FileTypes:  []string{"7z"},
		},
		{
			Name:       "PDF Document",
			MagicBytes: []byte{0x25, 0x50, 0x44, 0x46},
			FileTypes:  []string{"pdf"},
		},
		{
			Name:       "PGP Binary",
			MagicBytes: []byte{0x85},
			FileTypes:  []string{"pgp", "gpg"},
		},
		{
			Name:       "PGP Encrypted Data",
			MagicBytes: []byte{0x8C},
			FileTypes:  []string{"pgp", "gpg"},
		},
		{
			Name:       "PGP Compressed",
			MagicBytes: []byte{0xC4},
			FileTypes:  []string{"pgp", "gpg"},
		},
	}
}

func (lib *EncryptionFeatureLibrary) GetFeatures() []EncryptionFeature {
	return lib.features
}

func (lib *EncryptionFeatureLibrary) DetectFileType(data []byte) string {
	for _, feature := range lib.features {
		if bytes.HasPrefix(data, feature.MagicBytes) {
			if len(feature.FileTypes) > 0 {
				return feature.FileTypes[0]
			}
			return feature.Name
		}
	}
	return ""
}

type EncryptionDetector struct {
	featureLibrary *EncryptionFeatureLibrary
}

// defaultEncryptionDetector 包级单例：构造期一次性初始化 9 元素 feature 库，
// 避免每次 ExtractFile 重新分配。CheckEncryption 内部无状态，可安全共享。
var defaultEncryptionDetector = NewEncryptionDetector()

func NewEncryptionDetector() *EncryptionDetector {
	return &EncryptionDetector{
		featureLibrary: NewEncryptionFeatureLibrary(),
	}
}

func (d *EncryptionDetector) GetFeatureLibrary() *EncryptionFeatureLibrary {
	return d.featureLibrary
}

func (d *EncryptionDetector) readFileHeader(filePath string, size int) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	data := make([]byte, size)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return data[:n], nil
}

func (d *EncryptionDetector) CheckEncryption(filePath string) int {
	isEncrypted, _, _ := d.DetectEncryption(filePath)
	if isEncrypted {
		return 1
	}
	return 0
}

func (d *EncryptionDetector) DetectEncryption(filePath string) (bool, string, error) {
	header, err := d.readFileHeader(filePath, 512)
	if err != nil {
		return false, "", err
	}

	if len(header) < 4 {
		return false, "", nil
	}

	if bytes.HasPrefix(header, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
		return d.detectOLEEncryption(filePath, header)
	}

	if bytes.HasPrefix(header, []byte{0x50, 0x4B, 0x03, 0x04}) {
		return d.detectZipEncryption(header)
	}

	if bytes.HasPrefix(header, []byte{0x25, 0x50, 0x44, 0x46}) {
		return d.detectPDFEncryption(filePath)
	}

	if bytes.HasPrefix(header, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		return d.detect7zEncryption(filePath)
	}

	if bytes.HasPrefix(header, []byte{0x52, 0x61, 0x72, 0x21}) {
		return d.detectRAREncryption(header)
	}

	if d.isPGPFile(header) {
		return true, "PGP", nil
	}

	return false, "", nil
}

func (d *EncryptionDetector) detectOLEEncryption(filePath string, header []byte) (bool, string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))

	switch ext {
	case "doc", "dot", "wps", "wpt":
		return d.checkWordEncryption(filePath, header)
	case "xls", "xlt", "et", "ett":
		return d.checkExcelEncryption(filePath, header)
	case "ppt", "pot", "pps", "dps", "dpt":
		return d.checkPowerPointEncryption(filePath, header)
	case "docx", "dotx", "docm", "dotm", "xlsx", "xltx", "xlsm", "xltm", "pptx", "potx", "pptm", "potm", "ppsx", "ppsm", "xlsb":
		return d.checkOOXMLEncryption(filePath, header)
	case "vsd", "vss", "vst":
		return false, "Visio", nil
	default:
		return d.checkOOXMLEncryption(filePath, header)
	}
}

func (d *EncryptionDetector) checkWordEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "Word", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	sectorSize := int(1 << binary.LittleEndian.Uint16(header[0x1E:0x20]))
	err = parseOLEDirectory(file, header, 0, func(name string, entry []byte) error {
		if name != "WordDocument" {
			return nil
		}
		startSector := binary.LittleEndian.Uint32(entry[116:120])
		if _, err := file.Seek(int64(startSector)*int64(sectorSize), 0); err != nil {
			return err
		}
		fibHeader := make([]byte, 12)
		if _, err := io.ReadFull(file, fibHeader); err != nil {
			return err
		}
		wIdent := binary.LittleEndian.Uint16(fibHeader[0:2])
		if wIdent == 0xECA5 {
			return errEncrypted
		}
		return errNotEncrypted
	})
	if err != nil {
		if errors.Is(err, errEncrypted) {
			return true, "Word", nil
		}
		if errors.Is(err, errNotEncrypted) {
			return false, "Word", nil
		}
		return false, "Word", err
	}
	return false, "Word", nil
}

func (d *EncryptionDetector) checkExcelEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "Excel", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	sectorSize := int(1 << binary.LittleEndian.Uint16(header[0x1E:0x20]))
	err = parseOLEDirectory(file, header, 100, func(name string, entry []byte) error {
		if name != "Workbook" && name != "Book" {
			return nil
		}
		startSector := binary.LittleEndian.Uint32(entry[116:120])
		streamSize := binary.LittleEndian.Uint32(entry[120:124])
		if _, err := file.Seek(int64(startSector)*int64(sectorSize), 0); err != nil {
			return err
		}
		maxSearch := 2048
		if int(streamSize) < maxSearch {
			maxSearch = int(streamSize)
		}
		searchData := make([]byte, maxSearch)
		if _, err := io.ReadFull(file, searchData); err != nil {
			return err
		}
		for j := 0; j < len(searchData)-4; j++ {
			if binary.LittleEndian.Uint16(searchData[j:j+2]) == 0x002F {
				return errEncrypted
			}
		}
		return errNotEncrypted
	})
	if err != nil {
		if errors.Is(err, errEncrypted) {
			return true, "Excel", nil
		}
		if errors.Is(err, errNotEncrypted) {
			return false, "Excel", nil
		}
		return false, "Excel", err
	}
	return false, "Excel", nil
}

func (d *EncryptionDetector) checkPowerPointEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "PowerPoint", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	sectorSize := int(1 << binary.LittleEndian.Uint16(header[0x1E:0x20]))
	err = parseOLEDirectory(file, header, 0, func(name string, entry []byte) error {
		if name == "Current User" {
			startSector := binary.LittleEndian.Uint32(entry[116:120])
			streamSize := binary.LittleEndian.Uint32(entry[120:124])
			if streamSize >= 20 {
				if _, err := file.Seek(int64(startSector)*int64(sectorSize), 0); err != nil {
					return err
				}
				currentUserAtom := make([]byte, 20)
				if _, err := io.ReadFull(file, currentUserAtom); err != nil {
					return err
				}
				if binary.LittleEndian.Uint32(currentUserAtom[16:20]) == 0xE391C05F {
					return errEncrypted
				}
			}
			return nil
		}
		if strings.Contains(name, "Encrypted") {
			return errEncrypted
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errEncrypted) {
			return true, "PowerPoint", nil
		}
		return false, "PowerPoint", err
	}
	return false, "PowerPoint", nil
}

func (d *EncryptionDetector) checkOOXMLEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "Office", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	err = parseOLEDirectory(file, header, 100, func(name string, entry []byte) error {
		if strings.Contains(name, "EncryptionInfo") || strings.Contains(name, "EncryptedPackage") {
			return errEncrypted
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errEncrypted) {
			return true, "Office", nil
		}
		return false, "Office", err
	}

	// OOXML 特化：扩展名兜底（即使目录里没看到 EncryptionInfo 也按加密处理）
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext == "docx" || ext == "xlsx" || ext == "pptx" || ext == "docm" || ext == "xlsm" || ext == "pptm" || ext == "xlsb" {
		return true, "Office", nil
	}
	return false, "Office", nil
}

// errEncrypted / errNotEncrypted 是 parseOLEDirectory 回调约定的哨兵：
//
//	返回 errEncrypted   → 命中并认定为加密
//	返回 errNotEncrypted → 命中但非加密
//	真正的 IO 错误（Seeker/Read 失败）直接透传
//
// 回调返回非 nil 即终止迭代。
var (
	errEncrypted    = errors.New("ole entry indicates encryption")
	errNotEncrypted = errors.New("ole entry indicates non-encryption")
)

// parseOLEDirectory 通用 OLE 复合文档目录解析：给回调 (name, 128 字节 entry)；
// 回调返回 nil 继续迭代；返回 err 立即终止并把 err 透传给调用方。
//
// maxEntries=0  → 读满整个 sector（Word/PPT 风格，每 128 字节一个 entry）
// maxEntries>0  → 限条数（Excel/OOXML 风格）
func parseOLEDirectory(file *os.File, header []byte, maxEntries int, callback func(name string, entry []byte) error) error {
	sectorSize := int(1 << binary.LittleEndian.Uint16(header[0x1E:0x20]))
	firstDirSector := binary.LittleEndian.Uint32(header[0x3C:0x40])
	dirOffset := int64(firstDirSector) * int64(sectorSize)
	if _, err := file.Seek(dirOffset, 0); err != nil {
		return err
	}

	if maxEntries > 0 {
		entry := make([]byte, 128)
		for i := 0; i < maxEntries; i++ {
			n, rerr := io.ReadFull(file, entry)
			if rerr != nil || n < 128 {
				break
			}
			name, ok := decodeOLEEntryName(entry)
			if !ok {
				continue
			}
			if err := callback(name, entry); err != nil {
				return err
			}
		}
		return nil
	}

	// 整个 sector 风格
	dirEntries := make([]byte, sectorSize)
	n, rerr := io.ReadFull(file, dirEntries)
	if rerr != nil || n < sectorSize {
		return nil
	}
	for offset := 0; offset+128 <= sectorSize; offset += 128 {
		entry := dirEntries[offset : offset+128]
		name, ok := decodeOLEEntryName(entry)
		if !ok {
			continue
		}
		if err := callback(name, entry); err != nil {
			return err
		}
	}
	return nil
}

// decodeOLEEntryName 从 128 字节 OLE 目录项解出 name；空名/超长/损坏返回 ok=false。
func decodeOLEEntryName(entry []byte) (string, bool) {
	if len(entry) < 128 {
		return "", false
	}
	nameLen := binary.LittleEndian.Uint16(entry[64:66])
	if nameLen == 0 || nameLen > 64 {
		return "", false
	}
	u16 := make([]uint16, nameLen/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(entry[i*2 : i*2+2])
	}
	name := strings.TrimRight(string(utf16.Decode(u16)), "\x00")
	if name == "" {
		return "", false
	}
	return name, true
}

func (d *EncryptionDetector) detectZipEncryption(header []byte) (bool, string, error) {
	if len(header) < 8 {
		return false, "ZIP", nil
	}

	flags := binary.LittleEndian.Uint16(header[6:8])
	if flags&0x0001 != 0 {
		return true, "ZIP", nil
	}

	return false, "ZIP", nil
}

func (d *EncryptionDetector) detect7zEncryption(filePath string) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "7Z", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	signature := make([]byte, 6)
	_, err = io.ReadFull(file, signature)
	if err != nil {
		return false, "7Z", nil
	}

	if !bytes.Equal(signature, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		return false, "7Z", nil
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return false, "7Z", fmt.Errorf("获取文件信息失败: %v", err)
	}

	fileSize := fileInfo.Size()
	if fileSize < 32 {
		return false, "7Z", nil
	}

	file.Seek(12, 0)

	nextHeaderOffset := make([]byte, 8)
	nextHeaderSize := make([]byte, 8)
	nextHeaderCRC := make([]byte, 4)

	_, err = io.ReadFull(file, nextHeaderOffset)
	if err != nil {
		return false, "7Z", nil
	}
	_, err = io.ReadFull(file, nextHeaderSize)
	if err != nil {
		return false, "7Z", nil
	}
	_, err = io.ReadFull(file, nextHeaderCRC)
	if err != nil {
		return false, "7Z", nil
	}

	offset := int64(binary.LittleEndian.Uint64(nextHeaderOffset))
	size := binary.LittleEndian.Uint64(nextHeaderSize)

	if size == 0 || offset == 0 {
		headerEndOffset := int64(32 + 2)
		file.Seek(headerEndOffset, 0)

		headerByte := make([]byte, 1)
		file.Read(headerByte)

		if len(headerByte) > 0 {
			headerType := headerByte[0]
			if headerType == 0x17 || headerType == 0x06 {
				return true, "7Z", nil
			}
		}

		return false, "7Z", nil
	}

	headerPos := 32 + offset
	if headerPos >= fileSize {
		return false, "7Z", nil
	}

	file.Seek(headerPos, 0)

	headerData := make([]byte, 1)
	_, err = io.ReadFull(file, headerData)
	if err != nil {
		return false, "7Z", nil
	}

	headerType := headerData[0]
	if headerType == 0x17 {
		return true, "7Z", nil
	}

	if headerType == 0x01 {
		headerSize := int(size)
		if headerSize > 0 && headerSize < 10000 {
			file.Seek(headerPos, 0)
			fullHeader := make([]byte, headerSize)
			_, err = io.ReadFull(file, fullHeader)
			if err == nil {
				encryptorID := []byte{0x06, 0xF1, 0x07, 0x01}
				if bytes.Contains(fullHeader, encryptorID) {
					return true, "7Z", nil
				}
			}
		}
	}

	return false, "7Z", nil
}

func (d *EncryptionDetector) detectRAREncryption(header []byte) (bool, string, error) {
	if len(header) < 20 {
		return false, "RAR", nil
	}

	if !bytes.HasPrefix(header, []byte{0x52, 0x61, 0x72, 0x21}) {
		return false, "RAR", nil
	}

	// RAR4 格式（magic 0x52 0x61 0x72 0x21 0x1A 0x07 0x00）：可从首部可靠判断加密
	// RAR5（version 0x01 0x00）：加密标志位在文件头深处，无法仅靠 512 字节首部判断，
	// 保守返回 false，依赖下游 extractor 报错。
	if header[7] == 0x00 && header[6] != 0x01 {
		if len(header) >= 44 {
			headerType := header[38]
			headerFlags := binary.LittleEndian.Uint16(header[40:42])

			if (headerType == 0x01 || headerType == 0x02) && (headerFlags&0x0001 != 0) {
				return true, "RAR", nil
			}
		}
	}

	return false, "RAR", nil
}

func (d *EncryptionDetector) detectPDFEncryption(filePath string) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "PDF", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	searchData := make([]byte, 8192)
	n, err := file.Read(searchData)
	if err != nil && err != io.EOF {
		return false, "PDF", fmt.Errorf("读取文件失败: %w", err)
	}
	searchData = searchData[:n]

	encryptKeywords := [][]byte{
		[]byte("/Encrypt"),
		[]byte("/EncryptMetadata"),
		[]byte("/Filter /Standard"),
	}

	for _, keyword := range encryptKeywords {
		if bytes.Contains(searchData, keyword) {
			return true, "PDF", nil
		}
	}

	return false, "PDF", nil
}

func (d *EncryptionDetector) isPGPFile(header []byte) bool {
	if len(header) < 1 {
		return false
	}

	for _, p := range pgpNegativePrefixes {
		if len(header) >= len(p) && bytes.HasPrefix(header, p) {
			return false
		}
	}

	for _, tag := range pgpPositiveTags {
		if header[0] == tag {
			return true
		}
	}

	if len(header) >= 2 {
		tag := header[0] & 0xC0
		if tag == 0x80 || tag == 0xC0 {
			packetTag := header[0] & 0x3F
			if packetTag >= 1 && packetTag <= 19 {
				if !isPGPPacketNegativeHeader(header) {
					return true
				}
			}
		}
	}

	if len(header) >= 100 {
		if bytes.Contains(header[:100], []byte("PGP")) {
			return true
		}
	} else if len(header) >= 3 && bytes.Contains(header, []byte("PGP")) {
		return true
	}

	if len(header) >= 14 && bytes.HasPrefix(header, []byte("-----BEGIN PGP")) {
		return true
	}

	return false
}

// pgpNegativePrefixes 一旦 header 以这些前缀开头，必然不是 PGP（OLE/ZIP/PDF/7Z/RAR/PNG/JPEG/GIF/BMP/TIFF/RIFF/OGG/FLAC/ID3/MP3/MKV/EXE/ELF/CLASS/PS）。
// 表驱动：原本 150 行 switch-case，30+ 重复项（0xFF 0xD8 0xFF 出现 2 次、0x47 0x49 0x46 出现 2 次、0x42 0x4D 出现 2 次、TIFF 字节序出现 2 次）。
var pgpNegativePrefixes = [][]byte{
	{0xD0, 0xCF, 0x11, 0xE0},
	{0x50, 0x4B, 0x03, 0x04},
	{0x25, 0x50, 0x44, 0x46},
	{0x37, 0x7A, 0xBC, 0xAF},
	{0x52, 0x61, 0x72, 0x21},
	{0x89, 0x50, 0x4E, 0x47},
	{0x8B, 0x4A, 0x4E, 0x47},
	{0x8A, 0x4D, 0x4E, 0x47},
	{0xFF, 0xD8, 0xFF},
	{0x47, 0x49, 0x46},
	{0x42, 0x4D},
	{0x49, 0x49, 0x2A, 0x00},
	{0x4D, 0x4D, 0x00, 0x2A},
	{0x52, 0x49, 0x46, 0x46},
	{0x4F, 0x67, 0x67, 0x53},
	{0x66, 0x4C, 0x61, 0x43},
	{0x49, 0x44, 0x33},
	{0xFF, 0xFB},
	{0xFF, 0xFA},
	{0xFF, 0xF3},
	{0x1A, 0x45, 0xDF, 0xA3},
	{0x30, 0x26, 0xB2, 0x75},
	{0x4D, 0x5A},
	{0x7F, 0x45, 0x4C, 0x46},
	{0xCA, 0xFE, 0xBA, 0xBE},
	{0x25, 0x21, 0x50, 0x53},
	{0x81, 0x80},
}

// pgpPositiveTags PGP 公开已知的 old-format packet 头字节：0x85 公钥加密会话密钥包、0x8C 加密数据包、0xC4 压缩数据包 等。
var pgpPositiveTags = []byte{0x85, 0x8C, 0xC4, 0xA8, 0x84, 0x8D, 0x95, 0x99}

// isPGPPacketNegativeHeader 二次排除：当首字节的 packet tag 命中范围（0x80-0xBF、0xC0-0xDF），
// 但第 2/3 字节表明这是 PNG/JNG/MNG/EXE/RAR/7Z 等已知格式时返回 true（排除 PGP）。
func isPGPPacketNegativeHeader(header []byte) bool {
	if len(header) < 3 {
		return false
	}
	secondByte := header[1]
	thirdByte := header[2]

	if secondByte == 0x50 || secondByte == 0x4A || secondByte == 0x4D || secondByte == 0x47 {
		return true
	}
	switch {
	case secondByte == 0x89 && thirdByte == 0x50,
		secondByte == 0x8A && thirdByte == 0x4D,
		secondByte == 0x8B && thirdByte == 0x4A,
		secondByte == 0x81 && thirdByte == 0x80,
		secondByte == 0x52 && thirdByte == 0x61,
		secondByte == 0x37 && thirdByte == 0x7A:
		return true
	}
	return false
}

func (d *EncryptionDetector) GetEncryptionInfo(filePath string) map[string]interface{} {
	info := make(map[string]interface{})
	info["file_path"] = filePath

	isEncrypted, encType, err := d.DetectEncryption(filePath)
	info["is_encrypted"] = isEncrypted
	info["encryption_type"] = encType
	if err != nil {
		info["error"] = err.Error()
	}

	header, err := d.readFileHeader(filePath, 32)
	if err == nil {
		info["header_hex"] = hex.EncodeToString(header)
		info["file_type"] = d.featureLibrary.DetectFileType(header)
	}

	return info
}
