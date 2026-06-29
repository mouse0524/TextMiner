package extractor

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"textminer/pkg/logger"
	"textminer/pkg/magika/magika"
	"time"
)

var (
	fileTypeDetector  *FileTypeDetector
	magikaInitialized bool
)

// isODTFile 通过检查ZIP文件内容来判断是否为ODT文件
func isODTFile(filePath string) bool {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return false
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "content.xml" || f.Name == "mimetype" {
			return true
		}
	}
	return false
}

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
	FileName     string `json:"file_name"`    // 文件名
	FileType     string `json:"file_type"`    // 文件类型（MIME类型）
	FileSize     int64  `json:"file_size"`    // 文件大小（字节）
	Status       string `json:"status"`       // 提取状态：success/failed
	Content      string `json:"content"`      // 提取到的文本内容
	ErrorMessage string `json:"error_msg"`    // 错误信息（如果有）
	IsEncrypt    int    `json:"is_encrypt"`   // 是否加密：1=加密，0=未加密或不支持的类型
	ExecuteTime  string `json:"execute_time"` // 执行时间（毫秒）
}

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
	FileTypeVsd      = "vsd"
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
	// 新增文件类型
	"apk":     true,
	"azw3":    true,
	"blend":   true,
	"c4d":     true,
	"catpart": true,
	"chm":     true,
	"daf":     true,
	"dbf":     true,
	"dcm":     true,
	"djvu":    true,
	"dsm":     true,
	"dwg":     true,
	"dws":     true,
	"dxf":     true,
	"eml":     true,
	"exe":     true,
	"fbx":     true,
	"in":      true,
	"jar":     true,
	"lrf":     true,
	"m3u":     true,
	"m3u8":    true,
	"max":     true,
	"mht":     true,
	"mhtml":   true,
	"prt":     true,
	"sldasm":  true,
	"sldprt":  true,
	"snb":     true,
	"stl":     true,
	"tex":     true,
	"vcf":     true,
	"x3d":     true,
	"xpi":     true,
	"xps":     true,
}

// inferFileTypeFromMime 根据MIME类型推断文件类型
func inferFileTypeFromMime(mimeType string) string {
	switch mimeType {
	case "audio/midi":
		return "mid"
	case "audio/wav":
		return "wav"
	case "audio/ogg":
		return "ogg"
	case "audio/mpeg":
		return "mp3"
	case "audio/x-8svx":
		return "8svx"
	case "audio/aac":
		return "aac"
	case "audio/ac3":
		return "ac3"
	case "audio/aiff":
		return "aiff"
	case "audio/amb":
		return "amb"
	case "audio/amr":
		return "amr"
	case "audio/basic":
		return "au"
	case "audio/x-avr":
		return "avr"
	case "audio/x-caf":
		return "caf"
	case "audio/x-cdda":
		return "cdda"
	case "audio/x-cvs":
		return "cvs"
	case "audio/x-cvu":
		return "cvu"
	case "audio/x-dts":
		return "dts"
	case "audio/x-dvms":
		return "dvms"
	case "audio/x-fap":
		return "fap"
	case "audio/flac":
		return "flac"
	case "audio/x-fssd":
		return "fssd"
	case "audio/x-gsrt":
		return "gsrt"
	case "audio/x-hcom":
		return "hcom"
	case "audio/x-htk":
		return "htk"
	case "audio/x-ima":
		return "ima"
	case "audio/x-ircam":
		return "ircam"
	case "audio/mp4":
		return "m4a"
	case "audio/x-m4r":
		return "m4r"
	case "audio/x-maud":
		return "maud"
	case "audio/x-mmf":
		return "mmf"
	case "audio/x-nist":
		return "nist"
	case "audio/opus":
		return "opus"
	case "audio/x-paf":
		return "paf"
	case "audio/PCMA":
		return "pcma"
	case "audio/PCMU":
		return "pcmu"
	case "audio/x-pvf":
		return "pvf"
	case "audio/x-pn-realaudio":
		return "ra"
	case "audio/x-sd2":
		return "sd2"
	case "audio/x-sln":
		return "sln"
	case "audio/x-smp":
		return "smp"
	case "audio/x-snd":
		return "snd"
	case "audio/x-sou":
		return "sou"
	case "audio/x-sph":
		return "sph"
	case "audio/x-speex":
		return "spx"
	case "audio/x-tta":
		return "tta"
	case "audio/x-txw":
		return "txw"
	case "audio/x-vms":
		return "vms"
	case "audio/x-voc":
		return "voc"
	case "audio/x-vox":
		return "vox"
	case "audio/x-w64":
		return "w64"
	case "audio/x-ms-wma":
		return "wma"
	case "audio/x-wavpack":
		return "wv"
	case "audio/x-wve":
		return "wve"
	case "video/3gpp":
		return "3gp"
	case "video/mp4":
		return "mp4"
	case "application/x-mscompress-szdd":
		return "mscompress"
	case "application/winhlp":
		return "hlp"
	default:
		return ""
	}
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
	ext = ext[1:]

	return NewExtractorByType(ext)
}

// NewExtractorByType 根据文件类型创建对应的提取器
func NewExtractorByType(fileType string) (Extractor, error) {
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
		return &TxtExtractor{}, nil
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
		return &CodeExtractor{}, nil
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
	case FileTypePng, FileTypeJpg, FileTypeJpeg, FileTypeBmp, FileTypeGif, FileTypeTiff, FileTypeTif:
		return NewImageExtractor()
	case "mid", "midi", "wav", "ogg", "oga", "ogx", "mp3", "8svx", "aac", "ac3", "aiff", "aif", "amb", "amr", "au", "avr", "caf", "cdda", "cvs", "cvsd", "cvu", "dts", "dvms", "fap", "flac", "fssd", "gsrt", "hcom", "htk", "ima", "ircam", "m4a", "m4b", "m4p", "m4r", "maud", "mmf", "mp2", "nist", "opus", "paf", "pcma", "pcmu", "prc", "pvf", "ra", "ram", "sd2", "sln", "smp", "snd", "sndr", "sndt", "sou", "sph", "spx", "tta", "txw", "vms", "voc", "vox", "w64", "wma", "wv", "wve":
		return NewAudioExtractor()
	case "swf", "mp4", "mpg", "wmv", "3g2", "3gp", "asf", "avi", "dat", "dv", "f4v", "flv", "hevc", "m2ts", "m2v", "m4v", "mjpeg", "mkv", "mov", "mpeg", "mts", "mxf", "ogv", "rm", "rmvb", "vob", "webm", "wtv":
		return NewVideoExtractor()
	case "mscompress", "hlp":
		return &UnsupportedExtractor{fileType: fileType}, nil
	case "apk", "azw3", "blend", "c4d", "catpart", "chm", "daf", "dbf", "dcm", "djvu", "dsm", "dwg", "dws", "dxf", "eml", "exe", "fbx", "in", "jar", "lrf", "m3u", "m3u8", "max", "mht", "mhtml", "prt", "sldasm", "sldprt", "snb", "stl", "tex", "vcf", "x3d", "xpi", "xps":
		return NewMimeOnlyExtractor()
	default:
		return &UnsupportedExtractor{fileType: fileType}, nil
	}
}

// PgpExtractor PGP文件提取器
type PgpExtractor struct{}

func (e *PgpExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	detector := GetFileTypeDetector()
	_, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil || mimeType == "" {
		mimeType = MapExtensionToMimeType(ext[1:])
	}

	return &ExtractResult{
		FileName:     filepath.Base(filePath),
		FileType:     mimeType,
		FileSize:     0,
		Status:       "failed",
		Content:      "",
		ErrorMessage: "文件已加密，无法提取内容",
		IsEncrypt:    1,
		ExecuteTime:  "0.0000",
	}, errors.New("文件已加密，无法提取内容")
}

// UnsupportedExtractor 不支持的文件类型提取器
type UnsupportedExtractor struct {
	fileType string
}

func (e *UnsupportedExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
	encryptionDetector := NewEncryptionDetector()
	isEncrypt := encryptionDetector.CheckEncryption(filePath)

	ext := strings.ToLower(filepath.Ext(filePath))

	detector := GetFileTypeDetector()
	_, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil || mimeType == "" {
		mimeType = MapExtensionToMimeType(ext[1:])
	}

	return &ExtractResult{
		FileName:     filepath.Base(filePath),
		FileType:     mimeType,
		FileSize:     0,
		Status:       "failed",
		Content:      "",
		ErrorMessage: "不支持的文件类型",
		IsEncrypt:    isEncrypt,
	}, fmt.Errorf("不支持的文件类型: %s", e.fileType)
}

// ExtractFile 统一的文件提取入口函数
func ExtractFile(filePath string, enableOcr bool) (*ExtractResult, error) {
	startTime := time.Now()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logger.Errorf("获取文件信息失败: %s, 错误: %v", filePath, err)
		duration := time.Since(startTime)
		return &ExtractResult{
			FileName:     filepath.Base(filePath),
			FileType:     "",
			FileSize:     0,
			Status:       "failed",
			Content:      "",
			ErrorMessage: fmt.Sprintf("获取文件信息失败: %v", err),
			IsEncrypt:    0,
			ExecuteTime:  fmt.Sprintf("%.4f", duration.Seconds()),
		}, err
	}
	fileSize := fileInfo.Size()

	encryptionDetector := NewEncryptionDetector()
	isEncrypt := encryptionDetector.CheckEncryption(filePath)

	// 使用FileTypeDetector来获取文件类型（使用Magika检测无后缀文件）
	detector := NewFileTypeDetector(false)
	detectedFileType, mimeType, err := detector.GetDetailedInfo(filePath)
	if err != nil {
		// 如果获取失败，使用文件扩展名作为文件类型
		ext := strings.ToLower(filepath.Ext(filePath))
		detectedFileType = ext
		if ext != "" {
			detectedFileType = ext[1:]
		}
		mimeType = MapExtensionToMimeType(detectedFileType)
	}
	fileType := detectedFileType

	// 如果Magika返回的文件类型不支持，尝试从MIME类型推断文件类型
	if !IsFileTypeSupported(fileType) && mimeType != "" {
		// 根据MIME类型推断文件扩展名
		inferredType := inferFileTypeFromMime(mimeType)
		if inferredType != "" && IsFileTypeSupported(inferredType) {
			fileType = inferredType
			logger.Infof("根据MIME类型推断文件类型: MIME=%s, 推断类型=%s", mimeType, fileType)
		}
	}

	noPasswordCheckTypes := map[string]bool{}

	if noPasswordCheckTypes[fileType] {
		isEncrypt = 0
	}

	detector = GetFileTypeDetector()

	detectedType, detectedMime, err := detector.GetDetailedInfo(filePath)
	if err != nil {
		logger.Warnf("检测文件类型失败: %s, 错误: %v, 将使用扩展名", filePath, err)
	} else {
		logger.Infof("文件类型检测: 文件=%s, 扩展名=%s, 检测类型=%s, MIME=%s",
			filePath, fileType, detectedType, detectedMime)

		// 只有当检测到的类型支持时才使用，否则保持原类型和MIME
		if detectedType != "" && detectedType != "unknown" && IsFileTypeSupported(detectedType) {
			fileType = detectedType
			mimeType = detectedMime
		} else if detectedType != "" && detectedType != "unknown" {
			// 检测到类型但不支持，回退到使用扩展名
			ext := strings.ToLower(filepath.Ext(filePath))
			if ext != "" {
				fallbackType := ext[1:]
				if IsFileTypeSupported(fallbackType) {
					logger.Warnf("检测类型不支持，回退到扩展名: %s -> %s", detectedType, fallbackType)
					fileType = fallbackType
					mimeType = MapExtensionToMimeType(fallbackType)
				}
			}
		}
	}

	if detectedMime == "application/vnd.oasis.opendocument.text" {
		fileType = "odt"
	}

	if fileType == "" || fileType == "unknown" || detectedMime == "application/octet-stream" {
		if isODTFile(filePath) {
			logger.Infof("通过ZIP内容检测到ODT文件: %s", filePath)
			fileType = "odt"
			if detectedMime == "application/octet-stream" {
				detectedMime = "application/vnd.oasis.opendocument.text"
			}
		}
	}

	// 如果最终类型仍不支持，回退到使用扩展名
	if !IsFileTypeSupported(fileType) {
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != "" {
			fallbackType := ext[1:]
			if IsFileTypeSupported(fallbackType) {
				logger.Warnf("检测类型不支持，回退到扩展名: %s -> %s", fileType, fallbackType)
				fileType = fallbackType
				mimeType = MapExtensionToMimeType(fallbackType)
			}
		}
	}

	if !IsFileTypeSupported(fileType) {
		logger.Warnf("不支持的文件类型: %s (检测到的类型)", fileType)

		duration := time.Since(startTime)
		return &ExtractResult{
			FileName:     filepath.Base(filePath),
			FileType:     mimeType,
			FileSize:     fileSize,
			Status:       "failed",
			Content:      "",
			ErrorMessage: fmt.Sprintf("文件类型不支持: %s", fileType),
			IsEncrypt:    isEncrypt,
			ExecuteTime:  fmt.Sprintf("%.4f", duration.Seconds()),
		}, errors.New("文件类型不支持")
	}

	if isEncrypt == 1 {
		logger.Infof("文件已加密，直接返回: 文件=%s, 类型=%s, MIME=%s", filePath, fileType, mimeType)
		duration := time.Since(startTime)
		return &ExtractResult{
			FileName:     filepath.Base(filePath),
			FileType:     mimeType,
			FileSize:     fileSize,
			Status:       "failed",
			Content:      "",
			ErrorMessage: "文件已加密，无法提取内容",
			IsEncrypt:    1,
			ExecuteTime:  fmt.Sprintf("%.4f", duration.Seconds()),
		}, errors.New("文件已加密，无法提取内容")
	}

	extractor, err := NewExtractorByType(fileType)
	if err != nil {
		logger.Errorf("创建提取器失败: 类型=%s, 错误: %v", fileType, err)
		duration := time.Since(startTime)
		return &ExtractResult{
			FileName:     filepath.Base(filePath),
			FileType:     mimeType,
			FileSize:     fileSize,
			Status:       "failed",
			Content:      "",
			ErrorMessage: err.Error(),
			IsEncrypt:    isEncrypt,
			ExecuteTime:  fmt.Sprintf("%.4f", duration.Seconds()),
		}, err
	}

	logger.Infof("开始提取: 文件=%s, 类型=%s, MIME=%s, OCR=%v, 加密=%v", filePath, fileType, mimeType, enableOcr, isEncrypt == 1)
	result, err := extractor.Extract(filePath, enableOcr)
	if err != nil {
		logger.Errorf("提取失败: 文件=%s, 类型=%s, 错误: %v", filePath, fileType, err)
		if strings.Contains(err.Error(), "加密") {
			isEncrypt = 1
		}
	} else {
		logger.Infof("提取完成: 文件=%s, 类型=%s, 状态=%s", filePath, fileType, result.Status)
	}

	if mimeType != "" {
		result.FileType = mimeType
	}
	result.FileSize = fileSize
	result.IsEncrypt = isEncrypt
	result.ExecuteTime = fmt.Sprintf("%.4f", time.Since(startTime).Seconds())

	return result, err
}
