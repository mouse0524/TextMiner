package extractor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"textminer/pkg/logger"
	"textminer/pkg/magika/magika"
	"time"
)

var (
	fileTypeDetector  *FileTypeDetector
	magikaInitialized bool
)

// isODTFile 已移除：原实现会打开整个 zip 扫描 content.xml/mimetype，与下方 mime-based
// detection + 扩展名 fallback 路径重复。ODT 文件可由 application/vnd.oasis.opendocument.text
// MIME 或 .odt 扩展名识别，magika 失败的场景也由 fallback 兜底。

// InitMagika initializes the Magika scanner for file type detection
func InitMagika(assetsDir string) error {
	if assetsDir == "" {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		assetsDir = filepath.Join(filepath.Dir(execPath), "models")
	}

	if err := magika.InitScanner(assetsDir); err != nil {
		return fmt.Errorf("init magika scanner: %w", err)
	}

	fileTypeDetector = NewFileTypeDetector(true)
	magikaInitialized = true
	logger.Infof("Magika scanner initialized with assets dir: %s", assetsDir)
	return nil
}

// GetFileTypeDetector returns the file type detector instance
func GetFileTypeDetector() *FileTypeDetector {
	if fileTypeDetector == nil {
		fileTypeDetector = NewFileTypeDetector(magikaInitialized)
	}
	return fileTypeDetector
}

// Extractor 定义统一的内容提取接口
type Extractor interface {
	Extract(filePath string, enableOcr bool) (*ExtractResult, error)
}

// ExtractResult 定义提取结果结构
type ExtractResult struct {
	FileName     string `json:"file_name"`         // 文件名
	FileType     string `json:"file_type"`         // 文件类型（MIME类型）
	FileSize     int64  `json:"file_size"`         // 文件大小（字节）
	Status       string `json:"status"`            // 提取状态：success/failed/skipped
	Content      string `json:"content"`           // 提取到的文本内容
	ErrorMessage string `json:"error_msg"`         // 错误信息（如果有）
	IsEncrypt    int    `json:"is_encrypt"`        // 是否加密：1=加密，0=未加密或不支持的类型
	ExecuteTime  string `json:"execute_time"`      // 执行时间（毫秒）
	Skipped      bool   `json:"skipped,omitempty"` // 是否跳过（如不支持内容提取）
}

// Status 提取结果状态常量
const (
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"
	StatusWarning   = "warning"
	StatusEncrypted = "encrypted"
)

// ErrEncrypted 哨兵错误：表示目标文件已加密，无法提取明文内容。
// 调用方可使用 errors.Is(err, ErrEncrypted) 判断加密场景。
var ErrEncrypted = errors.New("文件已加密")

// FileType 定义支持的文件类型
const (
	FileTypeDoc      = "doc"
	FileTypeDocx     = "docx"
	FileTypeDot      = "dot"
	FileTypeDotx     = "dotx"
	FileTypeDotm     = "dotm"
	FileTypeDocm     = "docm"
	FileTypeWps      = "wps"
	FileTypeWpt      = "wpt"
	FileTypePpt      = "ppt"
	FileTypePptx     = "pptx"
	FileTypePot      = "pot"
	FileTypePotx     = "potx"
	FileTypePotm     = "potm"
	FileTypePps      = "pps"
	FileTypePpsm     = "ppsm"
	FileTypePpsx     = "ppsx"
	FileTypePptm     = "pptm"
	FileTypeDps      = "dps"
	FileTypeDpt      = "dpt"
	FileTypeVsd      = "vsd" // VSD binary format unsupported; see NewExtractorByType
	FileTypeVsdx     = "vsdx"
	FileTypeXls      = "xls"
	FileTypeXlsx     = "xlsx"
	FileTypeXlsm     = "xlsm"
	FileTypeXlsb     = "xlsb"
	FileTypeXlt      = "xlt"
	FileTypeXltx     = "xltx"
	FileTypeXltm     = "xltm"
	FileTypeXlam     = "xlam"
	FileTypeEt       = "et"
	FileTypeEtt      = "ett"
	FileTypeOdt      = "odt"
	FileTypePdf      = "pdf"
	FileTypeTxt      = "txt"
	FileTypeLog      = "log"
	FileTypeCsv      = "csv"
	FileTypeIni      = "ini"
	FileTypeRtf      = "rtf"
	FileTypeCode     = "code"
	FileTypeGo       = "go"
	FileTypeJava     = "java"
	FileTypePy       = "py"
	FileTypePyw      = "pyw"
	FileTypeC        = "c"
	FileTypeCpp      = "cpp"
	FileTypeH        = "h"
	FileTypeHpp      = "hpp"
	FileTypeCc       = "cc"
	FileTypeCxx      = "cxx"
	FileTypeHh       = "hh"
	FileTypeHxx      = "hxx"
	FileTypeJs       = "js"
	FileTypeJsx      = "jsx"
	FileTypeTs       = "ts"
	FileTypeTsx      = "tsx"
	FileTypeHtml     = "html"
	FileTypeHtm      = "htm"
	FileTypeCss      = "css"
	FileTypeScss     = "scss"
	FileTypeSass     = "sass"
	FileTypeLess     = "less"
	FileTypePhp      = "php"
	FileTypeRs       = "rs"
	FileTypeSwift    = "swift"
	FileTypeKt       = "kt"
	FileTypeKts      = "kts"
	FileTypeScala    = "scala"
	FileTypeRb       = "rb"
	FileTypeVbs      = "vbs"
	FileTypePl       = "pl"
	FileTypePm       = "pm"
	FileTypeSql      = "sql"
	FileTypeXml      = "xml"
	FileTypeJson     = "json"
	FileTypeYaml     = "yaml"
	FileTypeYml      = "yml"
	FileTypeMd       = "md"
	FileTypeMarkdown = "markdown"
	FileTypeSh       = "sh"
	FileTypeBash     = "bash"
	FileTypeBat      = "bat"
	FileTypePs1      = "ps1"
	FileTypeZip      = "zip"
	FileTypeSevenZip = "7z"
	FileTypeRar      = "rar"
	FileTypeTar      = "tar"
	FileTypeGz       = "gz"
	FileTypeTgz      = "tgz"
	FileTypeTarGz    = "tar.gz"
	FileTypeBz2      = "bz2"
	FileTypeXz       = "xz"
	FileTypeTarBz2   = "tar.bz2"
	FileTypeTarXz    = "tar.xz"
	FileTypeRpm      = "rpm"
	FileTypeIso      = "iso"
	FileTypePgp      = "pgp"
	FileTypePng      = "png"
	FileTypeJpg      = "jpg"
	FileTypeJpeg     = "jpeg"
	FileTypeBmp      = "bmp"
	FileTypeGif      = "gif"
	FileTypeTiff     = "tiff"
	FileTypeTif      = "tif"
	FileTypeWebp     = "webp"
)

// SupportedFileTypes 支持的文件类型列表
var SupportedFileTypes = map[string]bool{
	FileTypeDoc:  true,
	FileTypeDocx: true,
	FileTypeDot:  true,
	FileTypeDotx: true,
	FileTypeDotm: true,
	FileTypeDocm: true,
	FileTypeWps:  true,
	FileTypeWpt:  true,
	FileTypePpt:  true,
	FileTypePptx: true,
	FileTypePot:  true,
	FileTypePotx: true,
	FileTypePotm: true,
	FileTypePps:  true,
	FileTypePpsm: true,
	FileTypePpsx: true,
	FileTypePptm: true,
	FileTypeDps:  true,
	FileTypeDpt:  true,
	FileTypeVsd:  true,
	FileTypeVsdx: true,
	FileTypeXls:  true,
	FileTypeXlsx: true,
	FileTypeXlsm: true,
	FileTypeXlsb: true,
	FileTypeXlt:  true,
	FileTypeXltx: true,
	FileTypeXltm: true,
	FileTypeXlam: true,
	FileTypeEt:   true,
	FileTypeEtt:  true,
	FileTypeOdt:  true,
	FileTypePdf:  true,
	FileTypeTxt:  true,
	FileTypeLog:  true,
	FileTypeCsv:  true,
	FileTypeIni:  true,
	FileTypeRtf:  true,
	FileTypeCode: true,
	FileTypeGo:   true,
	FileTypeJava: true,
	FileTypePy:   true,
	FileTypePyw:  true,
	FileTypeC:    true,
	FileTypeCpp:  true,
	FileTypeH:    true,
	FileTypeHpp:  true,
	FileTypeCc:   true,
	FileTypeCxx:  true,
	// 音频文件类型
	"mid":   true,
	"midi":  true,
	"wav":   true,
	"ogg":   true,
	"oga":   true,
	"ogx":   true,
	"mp3":   true,
	"8svx":  true,
	"aac":   true,
	"ac3":   true,
	"aiff":  true,
	"aif":   true,
	"amb":   true,
	"amr":   true,
	"au":    true,
	"avr":   true,
	"caf":   true,
	"cdda":  true,
	"cvs":   true,
	"cvsd":  true,
	"cvu":   true,
	"dts":   true,
	"dvms":  true,
	"fap":   true,
	"flac":  true,
	"fssd":  true,
	"gsrt":  true,
	"hcom":  true,
	"htk":   true,
	"ima":   true,
	"ircam": true,
	"m4a":   true,
	"m4b":   true,
	"m4p":   true,
	"m4r":   true,
	"maud":  true,
	"mmf":   true,
	"mp2":   true,
	"nist":  true,
	"opus":  true,
	"paf":   true,
	"pcma":  true,
	"pcmu":  true,
	"prc":   true,
	"pvf":   true,
	"ra":    true,
	"ram":   true,
	"sd2":   true,
	"sln":   true,
	"smp":   true,
	"snd":   true,
	"sndr":  true,
	"sndt":  true,
	"sou":   true,
	"sph":   true,
	"spx":   true,
	"tta":   true,
	"txw":   true,
	"vms":   true,
	"voc":   true,
	"vox":   true,
	"w64":   true,
	"wma":   true,
	"wv":    true,
	"wve":   true,
	// 视频文件类型
	"swf":   true,
	"mp4":   true,
	"mpg":   true,
	"wmv":   true,
	"3g2":   true,
	"3gp":   true,
	"asf":   true,
	"avi":   true,
	"dat":   true,
	"dv":    true,
	"f4v":   true,
	"flv":   true,
	"hevc":  true,
	"m2ts":  true,
	"m2v":   true,
	"m4v":   true,
	"mjpeg": true,
	"mkv":   true,
	"mov":   true,
	"mpeg":  true,
	"mts":   true,
	"mxf":   true,
	"ogv":   true,
	"rm":    true,
	"rmvb":  true,
	"vob":   true,
	"webm":  true,
	"wtv":   true,
	// 其他文件类型
	"mscompress":     true,
	"hlp":            true,
	FileTypeHh:       true,
	FileTypeHxx:      true,
	FileTypeJs:       true,
	FileTypeJsx:      true,
	FileTypeTsx:      true,
	FileTypeHtml:     true,
	FileTypeHtm:      true,
	FileTypeCss:      true,
	FileTypeScss:     true,
	FileTypeSass:     true,
	FileTypeLess:     true,
	FileTypePhp:      true,
	FileTypeRs:       true,
	FileTypeSwift:    true,
	FileTypeKt:       true,
	FileTypeKts:      true,
	FileTypeScala:    true,
	FileTypeRb:       true,
	FileTypeVbs:      true,
	FileTypePl:       true,
	FileTypePm:       true,
	FileTypeSql:      true,
	FileTypeXml:      true,
	FileTypeJson:     true,
	FileTypeYaml:     true,
	FileTypeYml:      true,
	FileTypeMd:       true,
	FileTypeMarkdown: true,
	FileTypeSh:       true,
	FileTypeBash:     true,
	FileTypeBat:      true,
	FileTypePs1:      true,
	FileTypeZip:      true,
	FileTypeSevenZip: true,
	FileTypeRar:      true,
	FileTypeTar:      true,
	FileTypeGz:       true,
	FileTypeTgz:      true,
	FileTypeTarGz:    true,
	FileTypeBz2:      true,
	FileTypeXz:       true,
	FileTypeTarBz2:   true,
	FileTypeTarXz:    true,
	FileTypeRpm:      true,
	FileTypeIso:      true,
	FileTypePgp:      true,
	FileTypePng:      true,
	FileTypeJpg:      true,
	FileTypeJpeg:     true,
	FileTypeBmp:      true,
	FileTypeGif:      true,
	FileTypeTiff:     true,
	FileTypeTif:      true,
	FileTypeWebp:     true,
}

// mimeToExtMap 反向 map：MIME 类型 -> 优先扩展名。O(1) 查表替代原先的 100+ case switch。
// 当多个扩展名共享同一 MIME 时，本表显式指定返回的"代表"扩展名，与原 switch 行为一致。
var mimeToExtMap = map[string]string{
	"audio/midi":                    "mid",
	"audio/wav":                     "wav",
	"audio/ogg":                     "ogg",
	"audio/mpeg":                    "mp3",
	"audio/x-8svx":                  "8svx",
	"audio/aac":                     "aac",
	"audio/ac3":                     "ac3",
	"audio/aiff":                    "aiff",
	"audio/amb":                     "amb",
	"audio/amr":                     "amr",
	"audio/basic":                   "au",
	"audio/x-avr":                   "avr",
	"audio/x-caf":                   "caf",
	"audio/x-cdda":                  "cdda",
	"audio/x-cvs":                   "cvs",
	"audio/x-cvu":                   "cvu",
	"audio/x-dts":                   "dts",
	"audio/x-dvms":                  "dvms",
	"audio/x-fap":                   "fap",
	"audio/flac":                    "flac",
	"audio/x-fssd":                  "fssd",
	"audio/x-gsrt":                  "gsrt",
	"audio/x-hcom":                  "hcom",
	"audio/x-htk":                   "htk",
	"audio/x-ima":                   "ima",
	"audio/x-ircam":                 "ircam",
	"audio/mp4":                     "m4a",
	"audio/x-m4r":                   "m4r",
	"audio/x-maud":                  "maud",
	"audio/x-mmf":                   "mmf",
	"audio/x-nist":                  "nist",
	"audio/opus":                    "opus",
	"audio/x-paf":                   "paf",
	"audio/PCMA":                    "pcma",
	"audio/PCMU":                    "pcmu",
	"audio/x-pvf":                   "pvf",
	"audio/x-pn-realaudio":          "ra",
	"audio/x-sd2":                   "sd2",
	"audio/x-sln":                   "sln",
	"audio/x-smp":                   "smp",
	"audio/x-snd":                   "snd",
	"audio/x-sou":                   "sou",
	"audio/x-sph":                   "sph",
	"audio/x-speex":                 "spx",
	"audio/x-tta":                   "tta",
	"audio/x-txw":                   "txw",
	"audio/x-vms":                   "vms",
	"audio/x-voc":                   "voc",
	"audio/x-vox":                   "vox",
	"audio/x-w64":                   "w64",
	"audio/x-ms-wma":                "wma",
	"audio/x-wavpack":               "wv",
	"audio/x-wve":                   "wve",
	"video/3gpp":                    "3gp",
	"video/mp4":                     "mp4",
	"application/x-mscompress-szdd": "mscompress",
	"application/winhlp":            "hlp",
}

// inferFileTypeFromMime 根据MIME类型推断文件类型（O(1) map 查表）。
func inferFileTypeFromMime(mimeType string) string {
	if ext, ok := mimeToExtMap[mimeType]; ok {
		return ext
	}
	return ""
}

// IsFileTypeSupported 检查文件类型是否支持
func IsFileTypeSupported(fileType string) bool {
	return SupportedFileTypes[fileType]
}

// NewExtractor 根据文件类型创建对应的提取器
func NewExtractor(filePath string) (Extractor, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return nil, errors.New("文件没有扩展名")
	}

	// 移除点号
	ext = strings.TrimPrefix(ext, ".")

	return NewExtractorByType(ext)
}

// NewExtractorByType 根据文件类型创建对应的提取器。
// 主体 switch-case 仅处理"有专门 dispatcher"的类型；高频但无专门逻辑的
// 类型（audio/video/mime-only）走 O(1) 查表，避免主 switch 膨胀。
func NewExtractorByType(fileType string) (Extractor, error) {
	if fn, ok := audioExtractorFactories[fileType]; ok {
		return fn()
	}
	if fn, ok := videoExtractorFactories[fileType]; ok {
		return fn()
	}

	switch fileType {
	case FileTypeDoc, FileTypeWps, FileTypeWpt, FileTypeDot:
		return &DocExtractor{}, nil
	case FileTypeDocx, FileTypeDotx, FileTypeDotm, FileTypeDocm:
		return &DocxExtractor{}, nil
	case FileTypePpt, FileTypeDps, FileTypeDpt, FileTypePot, FileTypePps:
		return &PptExtractor{}, nil
	case FileTypePptx, FileTypePotx, FileTypePotm, FileTypePpsm, FileTypePptm, FileTypePpsx, FileTypeVsdx:
		return &PptxExtractor{}, nil
	case FileTypeVsd:
		return &VsdExtractor{}, nil
	case FileTypeXls, FileTypeEt, FileTypeEtt, FileTypeXlt:
		return &XlsExtractor{}, nil
	case FileTypeXlsx, FileTypeXlsm, FileTypeXltx, FileTypeXltm, FileTypeXlam:
		return &XlsxExtractor{}, nil
	case FileTypeXlsb:
		return &XlsbExtractor{}, nil
	case FileTypeOdt:
		return &DocxExtractor{}, nil
	case FileTypePdf:
		return &PdfExtractor{}, nil
	case FileTypeTxt, FileTypeLog, FileTypeIni:
		return NewTxtExtractor(false), nil
	case FileTypeCsv:
		return &CsvExtractor{}, nil
	case FileTypeRtf:
		return &RtfExtractor{}, nil
	case FileTypeGo, FileTypeJava, FileTypePy, FileTypePyw, FileTypeC, FileTypeCpp, FileTypeH, FileTypeHpp,
		FileTypeCc, FileTypeCxx, FileTypeHh, FileTypeHxx, FileTypeJs, FileTypeJsx, FileTypeTs, FileTypeTsx,
		FileTypeHtml, FileTypeHtm, FileTypeCss, FileTypeScss, FileTypeSass, FileTypeLess, FileTypePhp,
		FileTypeRs, FileTypeSwift, FileTypeKt, FileTypeKts, FileTypeScala, FileTypeRb, FileTypeVbs, FileTypePl, FileTypePm,
		FileTypeSql, FileTypeXml, FileTypeJson, FileTypeYaml, FileTypeYml, FileTypeMd, FileTypeMarkdown,
		FileTypeSh, FileTypeBash, FileTypeBat, FileTypePs1:
		return NewTxtExtractor(true), nil
	case FileTypeZip:
		return NewZipExtractor(), nil
	case FileTypeSevenZip:
		return NewSevenZipExtractor(), nil
	case FileTypeRar:
		return NewRarExtractor(), nil
	case FileTypeTar:
		return NewTarExtractor(), nil
	case FileTypeGz:
		return NewGzExtractor(), nil
	case FileTypeTgz, FileTypeTarGz:
		return NewTarGzExtractor(), nil
	case FileTypeBz2:
		return NewBz2Extractor(), nil
	case FileTypeXz:
		return NewXzExtractor(), nil
	case FileTypeTarBz2:
		return NewTarBz2Extractor(), nil
	case FileTypeTarXz:
		return NewTarXzExtractor(), nil
	case FileTypeRpm:
		return NewRpmExtractor(), nil
	case FileTypeIso:
		return NewIsoExtractor(), nil
	case FileTypePgp:
		return &PgpExtractor{}, nil
	case FileTypePng, FileTypeJpg, FileTypeJpeg, FileTypeBmp, FileTypeGif, FileTypeTiff, FileTypeTif, FileTypeWebp:
		return NewImageExtractor()
	case "mscompress", "hlp":
		return &UnsupportedExtractor{fileType: fileType}, nil
	default:
		return &UnsupportedExtractor{fileType: fileType}, nil
	}
}

// audioExtractorFactories 音频格式 -> 提取器构造器。
// 60+ 种扩展名以 O(1) map 查表代替 O(N) switch-case。
// 闭包包装是因为 NewAudioExtractor 返回 *MimeOnlyExtractor（具体类型），
// 不能直接赋值给 func() (Extractor, error)。
var audioExtractorFactories = func() map[string]func() (Extractor, error) {
	m := make(map[string]func() (Extractor, error), 60)
	exts := []string{
		"mid", "midi", "wav", "ogg", "oga", "ogx", "mp3", "8svx", "aac", "ac3",
		"aiff", "aif", "amb", "amr", "au", "avr", "caf", "cdda", "cvs", "cvsd",
		"cvu", "dts", "dvms", "fap", "flac", "fssd", "gsrt", "hcom", "htk", "ima",
		"ircam", "m4a", "m4b", "m4p", "m4r", "maud", "mmf", "mp2", "nist", "opus",
		"paf", "pcma", "pcmu", "prc", "pvf", "ra", "ram", "sd2", "sln", "smp", "snd",
		"sndr", "sndt", "sou", "sph", "spx", "tta", "txw", "vms", "voc", "vox",
		"w64", "wma", "wv", "wve",
	}
	for _, e := range exts {
		m[e] = func() (Extractor, error) { return NewAudioExtractor() }
	}
	return m
}()

var videoExtractorFactories = func() map[string]func() (Extractor, error) {
	m := make(map[string]func() (Extractor, error), 30)
	exts := []string{
		"swf", "mp4", "mpg", "wmv", "3g2", "3gp", "asf", "avi", "dat", "dv",
		"f4v", "flv", "hevc", "m2ts", "m2v", "m4v", "mjpeg", "mkv", "mov", "mpeg",
		"mts", "mxf", "ogv", "rm", "rmvb", "vob", "webm", "wtv",
	}
	for _, e := range exts {
		m[e] = func() (Extractor, error) { return NewVideoExtractor() }
	}
	return m
}()

// PgpExtractor PGP文件提取器
type PgpExtractor struct{}

func (e *PgpExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	content, enc, err := extractPgpContent(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("PGP 提取失败: %v", err)
		return result, err
	}
	if enc {
		result.Status = StatusEncrypted
		result.IsEncrypt = 1
		result.ErrorMessage = "PGP 加密块：未提供密钥或检测到二进制 packet 头"
		return result, ErrEncrypted
	}
	result.Content = content
	return result, nil
}

// extractPgpContent 读取 PGP 文件并尝试提取可读 ASCII armor 内容。
// 返回 (content, encrypted, err)：
//   - content：解码后的明文或 ASCII armor 块全文
//   - encrypted：true 表示文件实际包含加密的 PGP packet（0x85/0xc0/0xc3/0xc6）
//   - err：读取失败时非 nil
//
// 大文件优化：先扫前 1MB 找二进制 PGP packet 头或 ASCII armor 标记；
// 命中则只返回匹配区间，避免 io.ReadAll 整个文件 + 二次 string() 拷贝。
const pgpScanWindow = 1 << 20 // 1 MB

func extractPgpContent(filePath string) (string, bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	// 1) sniff 前 64 字节判定二进制 PGP packet 头
	var head [64]byte
	n, _ := f.Read(head[:])
	if n >= 1 {
		tag := head[0]
		if tag == 0x85 || tag == 0xC0 || tag == 0xC3 || tag == 0xC6 {
			return "", true, nil
		}
	}

	// 2) 扫 1MB 窗口找 ASCII armor 块
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", false, err
	}
	const sniffSize = pgpScanWindow
	buf := make([]byte, sniffSize)
	totalRead, _ := io.ReadFull(f, buf)
	if totalRead <= 0 {
		return "", false, nil
	}
	window := buf[:totalRead]

	const beginMarker = "-----BEGIN PGP"
	const endMarker = "-----END PGP"
	if bi := bytes.Index(window, []byte(beginMarker)); bi >= 0 {
		if ei := bytes.Index(window[bi:], []byte(endMarker)); ei >= 0 {
			ei2 := bi + ei + len(endMarker)
			if nl := bytes.IndexByte(window[ei2:], '\n'); nl >= 0 {
				ei2 += nl + 1
			}
			return string(window[bi:ei2]), false, nil
		}
		// armor 块跨 1MB 边界：回退到 io.ReadAll 找 END
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", false, err
		}
		data, err := io.ReadAll(f)
		if err != nil {
			return "", false, err
		}
		text := string(data)
		if ei := strings.Index(text[bi:], endMarker); ei >= 0 {
			ei2 := bi + ei + len(endMarker)
			if nl := strings.IndexByte(text[ei2:], '\n'); nl >= 0 {
				ei2 += nl + 1
			}
			return text[bi:ei2], false, nil
		}
		return text[bi:], false, nil
	}

	// 3) 没有任何 PGP 标记：1MB 窗口当纯文本/密钥环返回
	return string(window), false, nil
}

// UnsupportedExtractor 不支持的文件类型提取器
type UnsupportedExtractor struct {
	fileType string
}

func (e *UnsupportedExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	// 包级单例：避免每个 UnsupportedExtractor 实例都构造 9 元素 feature 库。
	isEncrypt := defaultEncryptionDetector.CheckEncryption(filePath)

	result.Status = StatusFailed
	result.ErrorMessage = fmt.Sprintf("不支持的文件类型: %s", e.fileType)
	result.IsEncrypt = isEncrypt
	return result, fmt.Errorf("不支持的文件类型: %s", e.fileType)
}

// ExtractFile 统一的文件提取入口函数
func ExtractFile(filePath string, enableOcr bool) (*ExtractResult, error) {
	startTime := time.Now()

	ctx, err := extractFileValidate(filePath)
	if err != nil {
		return makeFailureResult(filePath, "", 0, 0, err.Error(), startTime), err
	}

	fileType, mimeType, isEncrypt, err := extractFileDetect(ctx)
	if err != nil {
		return makeFailureResult(filePath, mimeType, ctx.FileSize, isEncrypt, err.Error(), startTime), err
	}

	extractor, err := NewExtractorByType(fileType)
	if err != nil {
		logger.Errorf("创建提取器失败: 类型=%s, 错误: %v", fileType, err)
		return makeFailureResult(filePath, mimeType, ctx.FileSize, isEncrypt, err.Error(), startTime), err
	}

	logger.Infof("开始提取: 文件=%s, 类型=%s, MIME=%s, OCR=%v, 加密=%v",
		ctx.FilePath, fileType, mimeType, enableOcr, isEncrypt == 1)
	result, err := extractor.Extract(ctx.FilePath, enableOcr)
	if err != nil {
		logger.Errorf("提取失败: 文件=%s, 类型=%s, 错误: %v", ctx.FilePath, fileType, err)
		if errors.Is(err, ErrEncrypted) {
			isEncrypt = 1
		}
	} else {
		logger.Infof("提取完成: 文件=%s, 类型=%s, 状态=%s", ctx.FilePath, fileType, result.Status)
	}

	return extractFileFinalize(ctx, fileType, mimeType, isEncrypt, result, startTime), err
}

// extractFileValidate 第一段：路径校验 + Stat，构造 ExtractContext。
func extractFileValidate(filePath string) (*ExtractContext, error) {
	cleanPath, err := validateFilePath(filePath)
	if err != nil {
		logger.Errorf("路径校验失败: %s, 错误: %v", filePath, err)
		return nil, err
	}
	filePath = cleanPath

	info, err := os.Stat(filePath)
	if err != nil {
		logger.Errorf("获取文件信息失败: %s, 错误: %v", filePath, err)
		return nil, err
	}
	return prepareExtractContextWithInfo(filePath, info)
}

// extractFileDetect 第二段：加密检测 + 类型检测 + 5 次回退。
func extractFileDetect(ctx *ExtractContext) (fileType, mimeType string, isEncrypt int, err error) {
	// 包级单例：9 元素 feature 库一次性初始化，无需每文件 NewEncryptionDetector()
	isEncrypt = defaultEncryptionDetector.CheckEncryption(ctx.FilePath)

	// 复用 prepareExtractContextWithInfo 已经做过的 GetDetailedInfo 结果，
	// 避免对同一文件再次调用（Magika ONNX 推理代价高）。
	detectedFileType := ctx.DetectedFileType
	detectedMime := ctx.DetectedMime
	if detectedFileType == "" {
		detectedFileType = strings.TrimPrefix(strings.ToLower(ctx.Ext), ".")
	}
	if detectedMime == "" {
		detectedMime = MapExtensionToMimeType(detectedFileType)
	}
	mimeType = detectedMime
	fileType = detectedFileType

	logger.Infof("文件类型检测: 文件=%s, 扩展名=%s, 检测类型=%s, MIME=%s",
		ctx.FilePath, fileType, detectedFileType, detectedMime)

	if !IsFileTypeSupported(fileType) && mimeType != "" {
		if inferredType := inferFileTypeFromMime(mimeType); inferredType != "" && IsFileTypeSupported(inferredType) {
			fileType = inferredType
			logger.Infof("根据MIME类型推断文件类型: MIME=%s, 推断类型=%s", mimeType, fileType)
		}
	}

	if mimeType == "application/vnd.oasis.opendocument.text" {
		fileType = "odt"
	}

	// 文件类型仍 unknown 但扩展名是 .odt：扩展名 fallback 会在下面的循环中命中
	// （line 800 的 IsFileTypeSupported 检查）。不需要再调 isODTFile 打开 zip。
	// 历史 isODTFile 已被移除，避免 3 次 zip 打开的浪费。

	if !IsFileTypeSupported(fileType) {
		if fallbackType := strings.TrimPrefix(strings.ToLower(ctx.Ext), "."); IsFileTypeSupported(fallbackType) {
			logger.Warnf("检测类型不支持，回退到扩展名: %s -> %s", fileType, fallbackType)
			fileType = fallbackType
			mimeType = MapExtensionToMimeType(fallbackType)
		}
	}

	if !IsFileTypeSupported(fileType) {
		logger.Warnf("不支持的文件类型: %s (检测到的类型)", fileType)
		err = errors.New("文件类型不支持")
		return
	}
	if isEncrypt == 1 {
		err = errors.New("文件已加密，无法提取内容")
		return
	}
	return
}

// extractFileFinalize 第三段：填充 ExecuteTime、FileSize、IsEncrypt、FileType。
func extractFileFinalize(ctx *ExtractContext, fileType, mimeType string, isEncrypt int, result *ExtractResult, startTime time.Time) *ExtractResult {
	if mimeType != "" {
		result.FileType = mimeType
	}
	result.FileSize = ctx.FileSize
	result.IsEncrypt = isEncrypt
	result.ExecuteTime = strconv.FormatFloat(time.Since(startTime).Seconds(), 'f', 4, 64)
	return result
}

// makeFailureResult 构造统一的失败结果：避免每个错误分支重复 8 行模板代码。
func makeFailureResult(filePath, mimeType string, fileSize int64, isEncrypt int, errMsg string, startTime time.Time) *ExtractResult {
	return &ExtractResult{
		FileName:     filepath.Base(filePath),
		FileType:     mimeType,
		FileSize:     fileSize,
		Status:       StatusFailed,
		Content:      "",
		ErrorMessage: errMsg,
		IsEncrypt:    isEncrypt,
		ExecuteTime:  strconv.FormatFloat(time.Since(startTime).Seconds(), 'f', 4, 64),
	}
}
