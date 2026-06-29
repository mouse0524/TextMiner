package extractor

import (
	"fmt"
	"path/filepath"
	"strings"

	"textminer/pkg/magika/magika"
)

// FileTypeDetector 文件真实类型检测器
type FileTypeDetector struct {
	useMagika bool
}

// NewFileTypeDetector creates a new FileTypeDetector with optional Magika support
func NewFileTypeDetector(useMagika bool) *FileTypeDetector {
	return &FileTypeDetector{useMagika: useMagika}
}

// InitMagika initializes the Magika scanner with the given assets directory
func (d *FileTypeDetector) InitMagika(assetsDir string) error {
	if err := magika.InitScanner(assetsDir); err != nil {
		return fmt.Errorf("init magika scanner: %w", err)
	}
	d.useMagika = true
	return nil
}

// DetectFileType 检测文件的真实类型
func (d *FileTypeDetector) DetectFileType(filePath string) (string, error) {
	if d.useMagika && magika.IsInitialized() {
		fileType, err := magika.DetectFileType(filePath)
		if err == nil && fileType != "" && fileType != "unknown" {
			return fileType, nil
		}
	}

	return d.DetectFileTypeByExtension(filePath), nil
}

// DetectFileTypeByExtension 根据文件扩展名获取文件类型（用于辅助判断）
func (d *FileTypeDetector) DetectFileTypeByExtension(filePath string) string {
	fileName := strings.ToLower(filepath.Base(filePath))

	doubleExts := []string{".tar.gz", ".tar.xz", ".tar.bz2"}
	for _, doubleExt := range doubleExts {
		if strings.HasSuffix(fileName, doubleExt) {
			return strings.TrimPrefix(doubleExt, ".")
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	return strings.TrimPrefix(ext, ".")
}

// GetDetailedInfo 获取文件类型的详细信息
func (d *FileTypeDetector) GetDetailedInfo(filePath string) (string, string, error) {
	var fileType, mimeType string

	extType := d.DetectFileTypeByExtension(filePath)

	officeExtensions := map[string]bool{
		"doc": true, "dot": true, "wps": true, "wpt": true,
		"docx": true, "dotx": true, "dotm": true, "docm": true,
		"xls": true, "xlt": true, "et": true, "ett": true, "xlsb": true,
		"xlsx": true, "xlsm": true, "xltx": true, "xltm": true, "xlam": true,
		"ppt": true, "pot": true, "pps": true, "dps": true, "dpt": true, "vsd": true,
		"pptx": true, "potx": true, "potm": true, "ppsm": true, "pptm": true, "ppsx": true, "vsdx": true,
		"odt": true,
	}

	if officeExtensions[extType] {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	videoExtensions := map[string]bool{
		"swf": true, "mp4": true, "mpg": true, "wmv": true, "3g2": true, "3gp": true, "asf": true, "avi": true, "dat": true,
		"dv": true, "f4v": true, "flv": true, "hevc": true, "m2ts": true, "m2v": true, "m4v": true, "mjpeg": true,
		"mkv": true, "mov": true, "mpeg": true, "mts": true, "mxf": true, "ogv": true, "rm": true, "rmvb": true,
		"vob": true, "webm": true, "wtv": true,
	}

	if videoExtensions[extType] {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	audioExtensions := map[string]bool{
		"mid": true, "midi": true, "wav": true, "ogg": true, "oga": true, "ogx": true, "mp3": true, "8svx": true, "aac": true, "ac3": true,
		"aiff": true, "aif": true, "amb": true, "amr": true, "au": true, "avr": true, "caf": true, "cdda": true,
		"cvs": true, "cvsd": true, "cvu": true, "dts": true, "dvms": true, "fap": true, "flac": true, "fssd": true,
		"gsrt": true, "hcom": true, "htk": true, "ima": true, "ircam": true, "m4a": true, "m4b": true, "m4p": true,
		"m4r": true, "maud": true, "mmf": true, "mp2": true, "nist": true, "opus": true, "paf": true, "pcma": true,
		"pcmu": true, "prc": true, "pvf": true, "ra": true, "ram": true, "sd2": true, "sln": true, "smp": true,
		"snd": true, "sndr": true, "sndt": true, "sou": true, "sph": true, "spx": true, "tta": true, "txw": true,
		"vms": true, "voc": true, "vox": true, "w64": true, "wma": true, "wv": true, "wve": true,
	}

	if audioExtensions[extType] {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	if d.useMagika && magika.IsInitialized() {
		ct, err := magika.GetContentType(filePath)
		if err == nil {
			fileType = magika.MapContentTypeToFileType(ct)
			mimeType = ct.MimeType
			// 检查Magika返回的文件类型是否支持
			if !IsFileTypeSupported(fileType) {
				// 如果不支持，回退到使用文件扩展名
				fileType = extType
				mimeType = MapExtensionToMimeType(fileType)
			}
			return fileType, mimeType, nil
		}
	}

	fileType = extType
	mimeType = MapExtensionToMimeType(extType)
	return fileType, mimeType, nil
}

// MapExtensionToMimeType 将文件扩展名映射到MIME类型
func MapExtensionToMimeType(ext string) string {
	switch ext {
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "odt":
		return "application/vnd.oasis.opendocument.text"
	case "xls":
		return "application/vnd.ms-excel"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "xlsb":
		return "application/vnd.ms-excel.sheet.binary.macroEnabled.12"
	case "xlt":
		return "application/vnd.ms-excel"
	case "xltx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.template"
	case "xltm":
		return "application/vnd.ms-excel.template.macroEnabled.12"
	case "xlam":
		return "application/vnd.ms-excel.addin.macroEnabled.12"
	case "ppt":
		return "application/vnd.ms-powerpoint"
	case "pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case "ppsx":
		return "application/vnd.openxmlformats-officedocument.presentationml.slideshow"
	case "vsd":
		return "application/vnd.visio"
	case "vsdx":
		return "application/vnd.visio"
	case "pdf":
		return "application/pdf"
	case "rtf":
		return "application/rtf"
	case "txt":
		return "text/plain"
	case "log":
		return "text/plain"
	case "csv":
		return "text/csv"
	case "ini":
		return "text/plain"
	case "zip":
		return "application/zip"
	case "7z":
		return "application/x-7z-compressed"
	case "rar":
		return "application/vnd.rar"
	case "tar":
		return "application/x-tar"
	case "gz":
		return "application/gzip"
	case "bz2":
		return "application/x-bzip2"
	case "xz":
		return "application/x-xz"
	case "tar.bz2":
		return "application/x-tar-bz2"
	case "tar.xz":
		return "application/x-tar-xz"
	case "rpm":
		return "application/x-rpm"
	case "iso":
		return "application/x-iso9660-image"
	case "tgz", "tar.gz":
		return "application/x-tar-gz"
	case "exe":
		return "application/vnd.microsoft.portable-executable"
	case "dll":
		return "application/vnd.microsoft.portable-executable"
	case "sys":
		return "application/vnd.microsoft.portable-executable"
	case "so":
		return "application/x-sharedlib"
	case "dylib":
		return "application/x-mach-binary"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "bmp":
		return "image/bmp"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "html", "htm":
		return "text/html"
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "ts":
		return "application/typescript"
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "yaml", "yml":
		return "application/x-yaml"
	case "mid", "midi":
		return "audio/midi"
	case "wav":
		return "audio/wav"
	case "ogg", "oga", "ogx":
		return "audio/ogg"
	case "mp3":
		return "audio/mpeg"
	case "8svx":
		return "audio/x-8svx"
	case "aac":
		return "audio/aac"
	case "ac3":
		return "audio/ac3"
	case "aiff", "aif":
		return "audio/aiff"
	case "amb":
		return "audio/amb"
	case "amr":
		return "audio/amr"
	case "au":
		return "audio/basic"
	case "avr":
		return "audio/x-avr"
	case "caf":
		return "audio/x-caf"
	case "cdda":
		return "audio/x-cdda"
	case "cvs", "cvsd":
		return "audio/x-cvs"
	case "cvu":
		return "audio/x-cvu"
	case "dts":
		return "audio/x-dts"
	case "dvms":
		return "audio/x-dvms"
	case "fap":
		return "audio/x-fap"
	case "flac":
		return "audio/flac"
	case "fssd":
		return "audio/x-fssd"
	case "gsrt":
		return "audio/x-gsrt"
	case "hcom":
		return "audio/x-hcom"
	case "htk":
		return "audio/x-htk"
	case "ima":
		return "audio/x-ima"
	case "ircam":
		return "audio/x-ircam"
	case "m4a", "m4b", "m4p":
		return "audio/mp4"
	case "m4r":
		return "audio/x-m4r"
	case "maud":
		return "audio/x-maud"
	case "mmf":
		return "audio/x-mmf"
	case "mp2":
		return "audio/mpeg"
	case "nist":
		return "audio/x-nist"
	case "opus":
		return "audio/opus"
	case "paf":
		return "audio/x-paf"
	case "pcma":
		return "audio/PCMA"
	case "pcmu":
		return "audio/PCMU"
	case "prc":
		return "audio/x-prc"
	case "pvf":
		return "audio/x-pvf"
	case "ra", "ram":
		return "audio/x-pn-realaudio"
	case "sd2":
		return "audio/x-sd2"
	case "sln":
		return "audio/x-sln"
	case "smp":
		return "audio/x-smp"
	case "snd":
		return "audio/basic"
	case "sndr", "sndt":
		return "audio/x-snd"
	case "sou":
		return "audio/x-sou"
	case "sph":
		return "audio/x-sph"
	case "spx":
		return "audio/x-speex"
	case "tta":
		return "audio/x-tta"
	case "txw":
		return "audio/x-txw"
	case "vms":
		return "audio/x-vms"
	case "voc":
		return "audio/x-voc"
	case "vox":
		return "audio/x-vox"
	case "w64":
		return "audio/x-w64"
	case "wma":
		return "audio/x-ms-wma"
	case "wv":
		return "audio/x-wavpack"
	case "wve":
		return "audio/x-wve"
	case "swf":
		return "application/x-shockwave-flash"
	case "mp4":
		return "video/mp4"
	case "mpg", "mpeg":
		return "video/mpeg"
	case "wmv":
		return "video/x-ms-wmv"
	case "3g2":
		return "video/3gpp2"
	case "3gp":
		return "video/3gpp"
	case "asf":
		return "video/x-ms-asf"
	case "avi":
		return "video/x-msvideo"
	case "dat":
		return "video/mp2t"
	case "dv":
		return "video/dv"
	case "f4v":
		return "video/x-f4v"
	case "flv":
		return "video/x-flv"
	case "hevc":
		return "video/hevc"
	case "m2ts":
		return "video/mp2t"
	case "m2v":
		return "video/mpeg"
	case "m4v":
		return "video/mp4"
	case "mjpeg":
		return "video/mjpeg"
	case "mkv":
		return "video/x-matroska"
	case "mov":
		return "video/quicktime"
	case "mts":
		return "video/mp2t"
	case "mxf":
		return "application/mxf"
	case "ogv":
		return "video/ogg"
	case "rm":
		return "application/vnd.rn-realmedia"
	case "rmvb":
		return "application/vnd.rn-realmedia-vbr"
	case "vob":
		return "video/x-ms-vob"
	case "webm":
		return "video/webm"
	case "wtv":
		return "video/x-ms-wtv"
	case "mscompress":
		return "application/x-mscompress"
	case "hlp":
		return "application/winhlp"
	case "md":
		return "text/markdown"
	case "go":
		return "text/x-go"
	case "java":
		return "text/x-java-source"
	case "py":
		return "text/x-python"
	case "c":
		return "text/x-c"
	case "cpp":
		return "text/x-c++"
	case "h":
		return "text/x-c"
	case "hpp":
		return "text/x-c++"
	case "php":
		return "application/x-httpd-php"
	case "rb":
		return "text/x-ruby"
	case "vbs":
		return "text/vbscript"
	case "rs":
		return "text/x-rust"
	case "swift":
		return "text/x-swift"
	case "kt":
		return "text/x-kotlin"
	case "tsx":
		return "application/typescript"
	case "jsx":
		return "text/jsx"
	case "vue":
		return "text/x-vue"
	case "sql":
		return "application/sql"
	case "sh", "bash":
		return "application/x-sh"
	case "bat":
		return "application/x-bat"
	case "ps1":
		return "application/x-powershell"
	case "apk":
		return "application/vnd.android.package-archive"
	case "azw3":
		return "application/vnd.amazon.ebook"
	case "blend":
		return "application/x-blender"
	case "c4d":
		return "application/vnd.c4d"
	case "catpart":
		return "application/vnd.catia"
	case "chm":
		return "application/vnd.ms-htmlhelp"
	case "daf":
		return "application/x-daf"
	case "dbf":
		return "application/x-dbf"
	case "dcm":
		return "application/dicom"
	case "djvu":
		return "image/vnd.djvu"
	case "dsm":
		return "application/x-dsm"
	case "dwg":
		return "application/acad"
	case "dws":
		return "application/x-dws"
	case "dxf":
		return "application/dxf"
	case "eml":
		return "message/rfc822"
	case "fbx":
		return "application/x-fbx"
	case "in":
		return "text/plain"
	case "jar":
		return "application/java-archive"
	case "lrf":
		return "application/x-sony-bbeb"
	case "m3u":
		return "audio/x-mpegurl"
	case "m3u8":
		return "application/vnd.apple.mpegurl"
	case "max":
		return "application/x-3ds"
	case "mht", "mhtml":
		return "message/rfc822"
	case "prt":
		return "application/x-prt"
	case "sldasm":
		return "application/vnd.solidworks"
	case "sldprt":
		return "application/vnd.solidworks"
	case "snb":
		return "application/x-snb"
	case "stl":
		return "application/sla"
	case "tex":
		return "application/x-tex"
	case "vcf":
		return "text/vcard"
	case "x3d":
		return "model/x3d+xml"
	case "xpi":
		return "application/x-xpinstall"
	case "xps":
		return "application/vnd.ms-xpsdocument"
	default:
		return "application/octet-stream"
	}
}

// ValidateFileType 验证文件扩展名与真实类型是否匹配
func (d *FileTypeDetector) ValidateFileType(filePath, expectedType string) bool {
	detectedType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}

	return d.isTypeMatch(detectedType, expectedType)
}

// isTypeMatch 判断检测到的类型与期望类型是否匹配
func (d *FileTypeDetector) isTypeMatch(detectedType, expectedType string) bool {
	if detectedType == expectedType {
		return true
	}

	typeGroups := map[string][]string{
		"doc":    {"doc", "dot", "wps", "wpt"},
		"docx":   {"docx", "dotx", "dotm", "docm", "odt"},
		"xls":    {"xls", "xlt", "et", "ett", "xlsb"},
		"xlsx":   {"xlsx", "xlsm", "xltx", "xltm", "xlam"},
		"ppt":    {"ppt", "pot", "pps", "dps", "dpt"},
		"pptx":   {"pptx", "potx", "potm", "ppsm", "pptm", "ppsx", "vsdx"},
		"vsd":    {"vsd"},
		"pdf":    {"pdf"},
		"rtf":    {"rtf"},
		"txt":    {"text", "txt", "plain", "log", "csv", "ini"},
		"log":    {"text", "txt", "plain", "log"},
		"csv":    {"csv"},
		"ini":    {"text", "txt", "plain", "ini"},
		"zip":    {"zip"},
		"7z":     {"7z"},
		"rar":    {"rar"},
		"tar":    {"tar"},
		"gz":     {"gz"},
		"tar.gz": {"tar.gz"},
		"bz2":    {"bz2"},
		"xz":     {"xz"},
		"rpm":    {"rpm"},
		"iso":    {"iso"},
		"tgz":    {"tgz", "tar.gz"},
		"png":    {"png"},
		"jpg":    {"jpg", "jpeg"},
		"bmp":    {"bmp"},
		"gif":    {"gif"},
		"webp":   {"webp"},
	}

	if group, ok := typeGroups[expectedType]; ok {
		for _, t := range group {
			if t == detectedType {
				return true
			}
		}
	}

	return false
}

// IsOfficeFile 判断是否是Office文件
func (d *FileTypeDetector) IsOfficeFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}

	officeTypes := []string{"doc", "docx", "xls", "xlsx", "ppt", "pptx", "wps", "dps", "et",
		"dot", "wpt", "xlt", "ett", "xlsb", "xlsm", "xltx", "xltm", "xlam", "odt", "pot", "pps", "ppsx", "dpt", "vsd", "vsdx"}
	for _, t := range officeTypes {
		if fileType == t {
			return true
		}
	}
	return false
}

// IsArchiveFile 判断是否是压缩文件
func (d *FileTypeDetector) IsArchiveFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}

	archiveTypes := []string{"zip", "7z", "rar", "tar", "gz", "bz2", "xz", "rpm", "iso", "tgz"}
	for _, t := range archiveTypes {
		if fileType == t {
			return true
		}
	}
	return false
}

// IsImageFile 判断是否是图片文件
func (d *FileTypeDetector) IsImageFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}

	imageTypes := []string{"png", "jpg", "jpeg", "bmp", "gif", "webp"}
	for _, t := range imageTypes {
		if fileType == t {
			return true
		}
	}
	return false
}

// IsTextFile 判断是否是文本文件
func (d *FileTypeDetector) IsTextFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}

	return fileType == "txt" || fileType == "text" || fileType == "plain" ||
		fileType == "log" || fileType == "csv" || fileType == "ini"
}

// GetFileCategory 获取文件分类
func (d *FileTypeDetector) GetFileCategory(filePath string) string {
	if d.IsOfficeFile(filePath) {
		return "office"
	}
	if d.IsArchiveFile(filePath) {
		return "archive"
	}
	if d.IsImageFile(filePath) {
		return "image"
	}
	if d.IsTextFile(filePath) {
		return "text"
	}

	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return "unknown"
	}

	switch fileType {
	case "pdf":
		return "pdf"
	case "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return "office"
	default:
		return "other"
	}
}

// FormatError 格式化错误信息
func (d *FileTypeDetector) FormatError(filePath string, err error) string {
	fileType, fileTypeErr := d.DetectFileType(filePath)
	if fileTypeErr != nil {
		return fmt.Sprintf("文件类型检测失败: %v", fileTypeErr)
	}

	switch fileType {
	case "doc", "docx":
		return fmt.Sprintf("打开Word文档失败: %v", err)
	case "xls", "xlsx":
		return fmt.Sprintf("打开Excel文件失败: %v", err)
	case "ppt", "pptx":
		return fmt.Sprintf("打开PowerPoint文件失败: %v", err)
	case "pdf":
		return fmt.Sprintf("打开PDF文件失败: %v", err)
	default:
		return fmt.Sprintf("打开文件失败: %v", err)
	}
}
