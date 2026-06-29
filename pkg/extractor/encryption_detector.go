package extractor

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

type EncryptionFeature struct {
	Name          string
	MagicBytes    []byte
	MagicOffset   int
	EncryptOffset int
	EncryptValue  []byte
	EncryptMask   []byte
	Description   string
	FileTypes     []string
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
			Name:        "OLE Compound Document",
			MagicBytes:  []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
			MagicOffset: 0,
			Description: "OLE复合文档格式（Office 97-2003及加密的OOXML文件）",
			FileTypes:   []string{"doc", "xls", "ppt", "docx", "xlsx", "pptx", "docm", "xlsm", "pptm", "xlsb", "wps", "et", "dps"},
		},
		{
			Name:          "ZIP Archive",
			MagicBytes:    []byte{0x50, 0x4B, 0x03, 0x04},
			MagicOffset:   0,
			EncryptOffset: 6,
			EncryptMask:   []byte{0x01, 0x00},
			EncryptValue:  []byte{0x01, 0x00},
			Description:   "ZIP压缩文件，加密标志位在偏移6处",
			FileTypes:     []string{"zip", "docx", "xlsx", "pptx", "odt", "ods", "odp"},
		},
		{
			Name:        "RAR5 Archive",
			MagicBytes:  []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00},
			MagicOffset: 0,
			Description: "RAR5压缩文件格式",
			FileTypes:   []string{"rar"},
		},
		{
			Name:        "RAR4 Archive",
			MagicBytes:  []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00},
			MagicOffset: 0,
			Description: "RAR4压缩文件格式",
			FileTypes:   []string{"rar"},
		},
		{
			Name:        "7Z Archive",
			MagicBytes:  []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},
			MagicOffset: 0,
			Description: "7-Zip压缩文件格式",
			FileTypes:   []string{"7z"},
		},
		{
			Name:        "PDF Document",
			MagicBytes:  []byte{0x25, 0x50, 0x44, 0x46},
			MagicOffset: 0,
			Description: "PDF文档格式",
			FileTypes:   []string{"pdf"},
		},
		{
			Name:        "PGP Binary",
			MagicBytes:  []byte{0x85},
			MagicOffset: 0,
			Description: "PGP公钥加密会话密钥包",
			FileTypes:   []string{"pgp", "gpg"},
		},
		{
			Name:        "PGP Encrypted Data",
			MagicBytes:  []byte{0x8C},
			MagicOffset: 0,
			Description: "PGP加密数据包",
			FileTypes:   []string{"pgp", "gpg"},
		},
		{
			Name:        "PGP Compressed",
			MagicBytes:  []byte{0xC4},
			MagicOffset: 0,
			Description: "PGP压缩数据包",
			FileTypes:   []string{"pgp", "gpg"},
		},
	}
}

func (lib *EncryptionFeatureLibrary) AddFeature(feature EncryptionFeature) {
	lib.features = append(lib.features, feature)
}

func (lib *EncryptionFeatureLibrary) GetFeatures() []EncryptionFeature {
	return lib.features
}

func (lib *EncryptionFeatureLibrary) DetectFileType(data []byte) string {
	for _, feature := range lib.features {
		if len(data) >= feature.MagicOffset+len(feature.MagicBytes) {
			if bytes.Equal(data[feature.MagicOffset:feature.MagicOffset+len(feature.MagicBytes)], feature.MagicBytes) {
				if len(feature.FileTypes) > 0 {
					return feature.FileTypes[0]
				}
				return feature.Name
			}
		}
	}
	return ""
}

type EncryptionDetector struct {
	featureLibrary *EncryptionFeatureLibrary
}

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

	sectorShift := binary.LittleEndian.Uint16(header[0x1E:0x20])
	sectorSize := int(1 << sectorShift)

	firstDirSector := binary.LittleEndian.Uint32(header[0x3C:0x40])

	dirOffset := int64(firstDirSector) * int64(sectorSize)
	file.Seek(dirOffset, 0)

	dirEntries := make([]byte, sectorSize)
	n, err := file.Read(dirEntries)
	if err != nil || n < sectorSize {
		return false, "Word", nil
	}

	for offset := 0; offset < sectorSize; offset += 128 {
		dirEntry := dirEntries[offset : offset+128]
		if len(dirEntry) < 128 {
			continue
		}

		nameLen := binary.LittleEndian.Uint16(dirEntry[64:66])
		if nameLen == 0 || nameLen > 64 {
			continue
		}

		entryName := make([]uint16, nameLen/2)
		for i := 0; i < int(nameLen/2); i++ {
			entryName[i] = binary.LittleEndian.Uint16(dirEntry[i*2 : i*2+2])
		}
		name := string(utf16.Decode(entryName))
		name = strings.TrimRight(name, "\x00")

		if name == "WordDocument" {
			startSector := binary.LittleEndian.Uint32(dirEntry[116:120])
			wordDocOffset := int64(startSector) * int64(sectorSize)

			file.Seek(wordDocOffset, 0)
			fibHeader := make([]byte, 12)
			_, err := io.ReadFull(file, fibHeader)
			if err != nil {
				continue
			}

			wIdent := binary.LittleEndian.Uint16(fibHeader[0:2])
			if wIdent == 0xECA5 {
				return true, "Word", nil
			}

			if wIdent == 0xA5DC || wIdent == 0xA59C {
				_ = binary.LittleEndian.Uint16(fibHeader[10:12])
				return false, "Word", nil
			}

			return false, "Word", nil
		}
	}

	return false, "Word", nil
}

func (d *EncryptionDetector) checkExcelEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "Excel", fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	sectorShift := binary.LittleEndian.Uint16(header[0x1E:0x20])
	sectorSize := int(1 << sectorShift)

	firstDirSector := binary.LittleEndian.Uint32(header[0x3C:0x40])

	dirOffset := int(firstDirSector) * sectorSize
	file.Seek(int64(dirOffset), 0)

	dirEntry := make([]byte, 128)
	for i := 0; i < 100; i++ {
		n, err := file.Read(dirEntry)
		if err != nil || n < 128 {
			break
		}

		nameLen := binary.LittleEndian.Uint16(dirEntry[64:66])
		if nameLen == 0 || nameLen > 64 {
			continue
		}

		entryName := make([]uint16, nameLen/2)
		for j := 0; j < int(nameLen/2); j++ {
			entryName[j] = binary.LittleEndian.Uint16(dirEntry[j*2 : j*2+2])
		}
		name := string(utf16.Decode(entryName))
		name = strings.TrimRight(name, "\x00")

		if name == "Workbook" || name == "Book" {
			startSector := binary.LittleEndian.Uint32(dirEntry[116:120])
			streamSize := binary.LittleEndian.Uint32(dirEntry[120:124])

			_ = streamSize

			streamOffset := int(startSector) * sectorSize
			file.Seek(int64(streamOffset), 0)

			maxSearch := 2048
			if maxSearch > int(streamSize) {
				maxSearch = int(streamSize)
			}

			searchData := make([]byte, maxSearch)
			file.Read(searchData)

			for j := 0; j < len(searchData)-4; j++ {
				recordID := binary.LittleEndian.Uint16(searchData[j : j+2])
				if recordID == 0x002F {
					return true, "Excel", nil
				}
			}

			return false, "Excel", nil
		}
	}

	return false, "Excel", nil
}

func (d *EncryptionDetector) checkPowerPointEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "PowerPoint", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	sectorShift := binary.LittleEndian.Uint16(header[0x1E:0x20])
	sectorSize := int(1 << sectorShift)

	firstDirSector := binary.LittleEndian.Uint32(header[0x3C:0x40])

	dirOffset := int(firstDirSector) * sectorSize
	file.Seek(int64(dirOffset), 0)

	dirEntries := make([]byte, sectorSize)
	n, err := file.Read(dirEntries)
	if err != nil || n < sectorSize {
		return false, "PowerPoint", nil
	}

	for offset := 0; offset < sectorSize; offset += 128 {
		dirEntry := dirEntries[offset : offset+128]
		if len(dirEntry) < 128 {
			continue
		}

		nameLen := binary.LittleEndian.Uint16(dirEntry[64:66])
		if nameLen == 0 || nameLen > 64 {
			continue
		}

		entryName := make([]uint16, nameLen/2)
		for i := 0; i < int(nameLen/2); i++ {
			entryName[i] = binary.LittleEndian.Uint16(dirEntry[i*2 : i*2+2])
		}
		name := string(utf16.Decode(entryName))
		name = strings.TrimRight(name, "\x00")

		if name == "Current User" {
			startSector := binary.LittleEndian.Uint32(dirEntry[116:120])
			streamSize := binary.LittleEndian.Uint32(dirEntry[120:124])
			streamOffset := int64(startSector) * int64(sectorSize)

			if streamSize >= 20 {
				file.Seek(streamOffset, 0)
				currentUserAtom := make([]byte, 20)
				_, err := io.ReadFull(file, currentUserAtom)
				if err != nil {
					continue
				}

				headerToken := binary.LittleEndian.Uint32(currentUserAtom[16:20])
				if headerToken == 0xE391C05F {
					return true, "PowerPoint", nil
				}
			}
		}

		if strings.Contains(name, "Encrypted") {
			return true, "PowerPoint", nil
		}
	}

	return false, "PowerPoint", nil
}

func (d *EncryptionDetector) checkOOXMLEncryption(filePath string, header []byte) (bool, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, "Office", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	sectorShift := binary.LittleEndian.Uint16(header[0x1E:0x20])
	sectorSize := int(1 << sectorShift)

	firstDirSector := binary.LittleEndian.Uint32(header[0x3C:0x40])

	dirOffset := int(firstDirSector) * sectorSize
	file.Seek(int64(dirOffset), 0)

	dirEntry := make([]byte, 128)
	for i := 0; i < 100; i++ {
		n, err := file.Read(dirEntry)
		if err != nil || n < 128 {
			break
		}

		nameLen := binary.LittleEndian.Uint16(dirEntry[64:66])
		if nameLen == 0 || nameLen > 64 {
			continue
		}

		entryName := make([]uint16, nameLen/2)
		for j := 0; j < int(nameLen/2); j++ {
			entryName[j] = binary.LittleEndian.Uint16(dirEntry[j*2 : j*2+2])
		}
		name := string(utf16.Decode(entryName))
		name = strings.TrimRight(name, "\x00")

		if strings.Contains(name, "EncryptionInfo") || strings.Contains(name, "EncryptedPackage") {
			return true, "Office", nil
		}
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext == "docx" || ext == "xlsx" || ext == "pptx" || ext == "docm" || ext == "xlsm" || ext == "pptm" || ext == "xlsb" {
		return true, "Office", nil
	}

	return false, "Office", nil
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
		encodedHeader := make([]byte, 20)
		n, err := io.ReadFull(file, encodedHeader)
		if err != nil || n < 2 {
			return false, "7Z", nil
		}

		_ = encodedHeader[0]
		_ = encodedHeader[1]

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

	// RAR5 format
	if header[6] == 0x01 && header[7] == 0x00 {
		if len(header) >= 32 {
			vint := header[8:]
			if len(vint) > 0 {
				extraSize := 0
				shift := 0
				for i := 0; i < len(vint) && i < 10; i++ {
					extraSize |= int(vint[i]&0x7F) << shift
					if vint[i]&0x80 == 0 {
						break
					}
					shift += 7
				}
				headerFlagsPos := 8 + extraSize + 4
				if headerFlagsPos+2 <= len(header) {
					headerFlags := binary.LittleEndian.Uint16(header[headerFlagsPos : headerFlagsPos+2])
					// RAR5: bit 0 is Volume flag, not encryption
					// We need to check for encryption in file headers
					// For now, return false as we cannot reliably detect RAR5 encryption from header only
					_ = headerFlags
				}
			}
		}
	} else if header[6] == 0x00 {
		// RAR4 format
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

	if len(header) >= 4 {
		if bytes.HasPrefix(header, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x50, 0x4B, 0x03, 0x04}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x25, 0x50, 0x44, 0x46}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x37, 0x7A, 0xBC, 0xAF}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x52, 0x61, 0x72, 0x21}) {
			return false
		}
	}

	if len(header) >= 8 {
		if bytes.HasPrefix(header, []byte{0x89, 0x50, 0x4E, 0x47}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x8B, 0x4A, 0x4E, 0x47}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x8A, 0x4D, 0x4E, 0x47}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0xFF, 0xD8, 0xFF}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x47, 0x49, 0x46}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x42, 0x4D}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x49, 0x49, 0x2A, 0x00}) || bytes.HasPrefix(header, []byte{0x4D, 0x4D, 0x00, 0x2A}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0xFF, 0xD8, 0xFF}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x47, 0x49, 0x46}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x42, 0x4D}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x49, 0x49, 0x2A, 0x00}) || bytes.HasPrefix(header, []byte{0x4D, 0x4D, 0x00, 0x2A}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x52, 0x49, 0x46, 0x46}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x4F, 0x67, 0x67, 0x53}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x66, 0x4C, 0x61, 0x43}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x49, 0x44, 0x33}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0xFF, 0xFB}) || bytes.HasPrefix(header, []byte{0xFF, 0xFA}) || bytes.HasPrefix(header, []byte{0xFF, 0xF3}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x1A, 0x45, 0xDF, 0xA3}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x30, 0x26, 0xB2, 0x75}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x4D, 0x5A}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x7F, 0x45, 0x4C, 0x46}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0xCA, 0xFE, 0xBA, 0xBE}) {
			return false
		}
		if bytes.HasPrefix(header, []byte{0x25, 0x21, 0x50, 0x53}) {
			return false
		}
		if header[0] == 0x81 && header[1] == 0x80 {
			return false
		}
	}

	pgpTags := []byte{0x85, 0x8C, 0xC4, 0xA8, 0x84, 0x8D, 0x95, 0x99}
	for _, tag := range pgpTags {
		if header[0] == tag {
			return true
		}
	}

	if len(header) >= 2 {
		tag := header[0] & 0xC0
		if tag == 0x80 || tag == 0xC0 {
			packetTag := header[0] & 0x3F
			if packetTag >= 1 && packetTag <= 19 {
				if len(header) >= 3 {
					secondByte := header[1]
					thirdByte := header[2]
					if secondByte == 0x50 || secondByte == 0x4A || secondByte == 0x4D || secondByte == 0x47 {
						return false
					}
					if secondByte == 0x89 && thirdByte == 0x50 {
						return false
					}
					if secondByte == 0x8A && thirdByte == 0x4D {
						return false
					}
					if secondByte == 0x8B && thirdByte == 0x4A {
						return false
					}
					if secondByte == 0x81 && thirdByte == 0x80 {
						return false
					}
					if secondByte == 0x52 && thirdByte == 0x61 {
						return false
					}
					if secondByte == 0x37 && thirdByte == 0x7A {
						return false
					}
				}
				return true
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


