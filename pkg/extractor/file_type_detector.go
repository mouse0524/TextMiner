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

// 类别集合：避免每次 GetDetailedInfo 重新构造，提升热路径性能
var (
	officeExtensionsMap = map[string]struct{}{
		"doc": {}, "dot": {}, "wps": {}, "wpt": {},
		"docx": {}, "dotx": {}, "dotm": {}, "docm": {},
		"xls": {}, "xlt": {}, "et": {}, "ett": {}, "xlsb": {},
		"xlsx": {}, "xlsm": {}, "xltx": {}, "xltm": {}, "xlam": {},
		"ppt": {}, "pot": {}, "pps": {}, "dps": {}, "dpt": {}, "vsd": {}, "vsdx": {},
		"odt": {},
	}
	archiveExtensionsMap = map[string]struct{}{
		"zip": {}, "7z": {}, "rar": {}, "tar": {}, "gz": {}, "bz2": {}, "xz": {}, "rpm": {}, "iso": {}, "tgz": {},
	}
	imageExtensionsMap = map[string]struct{}{
		"png": {}, "jpg": {}, "jpeg": {}, "bmp": {}, "gif": {}, "webp": {},
		// 较常见的图片扩展名
		"jpe": {}, "tif": {}, "tiff": {}, "ico": {}, "svg": {}, "psd": {},
		"heic": {}, "heif": {}, "tga": {}, "pcx": {}, "jp2": {}, "jpx": {},
		// 较冷门但仍常见的图像扩展名（OCR 通常不支持，会退化为元数据提取）
		"cur": {}, "dds": {}, "exr": {}, "eps": {}, "iff": {}, "jpf": {},
		"jng": {}, "mng": {}, "pbm": {}, "pcd": {}, "pgm": {}, "pnm": {},
		"ppm": {}, "psb": {}, "pxr": {}, "sct": {}, "wbmp": {}, "xpm": {},
	}
	videoExtensionsMap = map[string]struct{}{
		"swf": {}, "mp4": {}, "mpg": {}, "wmv": {}, "3g2": {}, "3gp": {}, "asf": {}, "avi": {}, "dat": {},
		"dv": {}, "f4v": {}, "flv": {}, "hevc": {}, "m2ts": {}, "m2v": {}, "m4v": {}, "mjpeg": {},
		"mkv": {}, "mov": {}, "mpeg": {}, "mts": {}, "mxf": {}, "ogv": {}, "rm": {}, "rmvb": {},
		"vob": {}, "webm": {}, "wtv": {},
	}
	audioExtensionsMap = map[string]struct{}{
		"mid": {}, "midi": {}, "wav": {}, "ogg": {}, "oga": {}, "ogx": {}, "mp3": {}, "8svx": {}, "aac": {}, "ac3": {},
		"aiff": {}, "aif": {}, "amb": {}, "amr": {}, "au": {}, "avr": {}, "caf": {}, "cdda": {},
		"cvs": {}, "cvsd": {}, "cvu": {}, "dts": {}, "dvms": {}, "fap": {}, "flac": {}, "fssd": {},
		"gsrt": {}, "hcom": {}, "htk": {}, "ima": {}, "ircam": {}, "m4a": {}, "m4b": {}, "m4p": {},
		"m4r": {}, "maud": {}, "mmf": {}, "mp2": {}, "nist": {}, "opus": {}, "paf": {}, "pcma": {},
		"pcmu": {}, "prc": {}, "pvf": {}, "ra": {}, "ram": {}, "sd2": {}, "sln": {}, "smp": {},
		"snd": {}, "sndr": {}, "sndt": {}, "sou": {}, "sph": {}, "spx": {}, "tta": {}, "txw": {},
		"vms": {}, "voc": {}, "vox": {}, "w64": {}, "wma": {}, "wv": {}, "wve": {},
	}
)

// GetDetailedInfo 获取文件类型的详细信息
func (d *FileTypeDetector) GetDetailedInfo(filePath string) (string, string, error) {
	var fileType, mimeType string

	extType := d.DetectFileTypeByExtension(filePath)

	if _, ok := officeExtensionsMap[extType]; ok {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	if _, ok := videoExtensionsMap[extType]; ok {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	if _, ok := audioExtensionsMap[extType]; ok {
		fileType = extType
		mimeType = MapExtensionToMimeType(extType)
		return fileType, mimeType, nil
	}

	if d.useMagika && magika.IsInitialized() {
		ct, err := magika.GetContentType(filePath)
		if err == nil {
			fileType = magika.MapContentTypeToFileType(ct)
			mimeType = ct.MimeType
			if !IsFileTypeSupported(fileType) {
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

// extToMimeMap 扩展名到 MIME 类型的预编译映射（替代原先 O(N) 的 switch-case）。
// 多个扩展名映射到同一 MIME 时分别列出，便于维护。
var extToMimeMap = map[string]string{
	"doc":        "application/msword",
	"docx":       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"odt":        "application/vnd.oasis.opendocument.text",
	"xls":        "application/vnd.ms-excel",
	"xlsx":       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xlsb":       "application/vnd.ms-excel.sheet.binary.macroEnabled.12",
	"xlt":        "application/vnd.ms-excel",
	"xltx":       "application/vnd.openxmlformats-officedocument.spreadsheetml.template",
	"xltm":       "application/vnd.ms-excel.template.macroEnabled.12",
	"xlam":       "application/vnd.ms-excel.addin.macroEnabled.12",
	"ppt":        "application/vnd.ms-powerpoint",
	"pptx":       "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"ppsx":       "application/vnd.openxmlformats-officedocument.presentationml.slideshow",
	"vsd":        "application/vnd.visio",
	"vsdx":       "application/vnd.visio",
	"pdf":        "application/pdf",
	"rtf":        "application/rtf",
	"txt":        "text/plain",
	"log":        "text/plain",
	"csv":        "text/csv",
	"ini":        "text/plain",
	"zip":        "application/zip",
	"7z":         "application/x-7z-compressed",
	"rar":        "application/vnd.rar",
	"tar":        "application/x-tar",
	"gz":         "application/gzip",
	"bz2":        "application/x-bzip2",
	"xz":         "application/x-xz",
	"tar.bz2":    "application/x-tar-bz2",
	"tar.xz":     "application/x-tar-xz",
	"rpm":        "application/x-rpm",
	"iso":        "application/x-iso9660-image",
	"tgz":        "application/x-tar-gz",
	"tar.gz":     "application/x-tar-gz",
	"exe":        "application/vnd.microsoft.portable-executable",
	"dll":        "application/vnd.microsoft.portable-executable",
	"sys":        "application/vnd.microsoft.portable-executable",
	"so":         "application/x-sharedlib",
	"dylib":      "application/x-mach-binary",
	"png":        "image/png",
	"jpg":        "image/jpeg",
	"jpeg":       "image/jpeg",
	"jpe":        "image/jpeg",
	"bmp":        "image/bmp",
	"gif":        "image/gif",
	"webp":       "image/webp",
	"tif":        "image/tiff",
	"tiff":       "image/tiff",
	"ico":        "image/x-icon",
	"svg":        "image/svg+xml",
	"psd":        "image/vnd.adobe.photoshop",
	"heic":       "image/heic",
	"heif":       "image/heif",
	"tga":        "image/x-tga",
	"pcx":        "image/x-pcx",
	"jp2":        "image/jp2",
	"jpx":        "image/jpx",
	"cur":        "image/x-icon",
	"dds":        "image/x-dds",
	"exr":        "image/x-exr",
	"eps":        "application/postscript",
	"iff":        "image/x-iff",
	"jpf":        "image/jpx",
	"jng":        "image/x-jng",
	"mng":        "video/x-mng",
	"pbm":        "image/x-portable-bitmap",
	"pcd":        "image/x-photo-cd",
	"pgm":        "image/x-portable-graymap",
	"pnm":        "image/x-portable-anymap",
	"ppm":        "image/x-portable-pixmap",
	"psb":        "image/vnd.adobe.photoshop",
	"pxr":        "image/x-pixar",
	"sct":        "image/x-sct",
	"wbmp":       "image/vnd.wap.wbmp",
	"xpm":        "image/x-xpm",
	"html":       "text/html",
	"htm":        "text/html",
	"css":        "text/css",
	"js":         "application/javascript",
	"ts":         "application/typescript",
	"tsx":        "application/typescript",
	"jsx":        "text/jsx",
	"vue":        "text/x-vue",
	"json":       "application/json",
	"xml":        "application/xml",
	"yaml":       "application/x-yaml",
	"yml":        "application/x-yaml",
	"mid":        "audio/midi",
	"midi":       "audio/midi",
	"wav":        "audio/wav",
	"ogg":        "audio/ogg",
	"oga":        "audio/ogg",
	"ogx":        "audio/ogg",
	"mp3":        "audio/mpeg",
	"mp2":        "audio/mpeg",
	"8svx":       "audio/x-8svx",
	"aac":        "audio/aac",
	"ac3":        "audio/ac3",
	"aiff":       "audio/aiff",
	"aif":        "audio/aiff",
	"amb":        "audio/amb",
	"amr":        "audio/amr",
	"au":         "audio/basic",
	"snd":        "audio/basic",
	"avr":        "audio/x-avr",
	"caf":        "audio/x-caf",
	"cdda":       "audio/x-cdda",
	"cvs":        "audio/x-cvs",
	"cvsd":       "audio/x-cvs",
	"cvu":        "audio/x-cvu",
	"dts":        "audio/x-dts",
	"dvms":       "audio/x-dvms",
	"fap":        "audio/x-fap",
	"flac":       "audio/flac",
	"fssd":       "audio/x-fssd",
	"gsrt":       "audio/x-gsrt",
	"hcom":       "audio/x-hcom",
	"htk":        "audio/x-htk",
	"ima":        "audio/x-ima",
	"ircam":      "audio/x-ircam",
	"m4a":        "audio/mp4",
	"m4b":        "audio/mp4",
	"m4p":        "audio/mp4",
	"m4r":        "audio/x-m4r",
	"maud":       "audio/x-maud",
	"mmf":        "audio/x-mmf",
	"nist":       "audio/x-nist",
	"opus":       "audio/opus",
	"paf":        "audio/x-paf",
	"pcma":       "audio/PCMA",
	"pcmu":       "audio/PCMU",
	"prc":        "audio/x-prc",
	"pvf":        "audio/x-pvf",
	"ra":         "audio/x-pn-realaudio",
	"ram":        "audio/x-pn-realaudio",
	"sd2":        "audio/x-sd2",
	"sln":        "audio/x-sln",
	"smp":        "audio/x-smp",
	"sndr":       "audio/x-snd",
	"sndt":       "audio/x-snd",
	"sou":        "audio/x-sou",
	"sph":        "audio/x-sph",
	"spx":        "audio/x-speex",
	"tta":        "audio/x-tta",
	"txw":        "audio/x-txw",
	"vms":        "audio/x-vms",
	"voc":        "audio/x-voc",
	"vox":        "audio/x-vox",
	"w64":        "audio/x-w64",
	"wma":        "audio/x-ms-wma",
	"wv":         "audio/x-wavpack",
	"wve":        "audio/x-wve",
	"swf":        "application/x-shockwave-flash",
	"mp4":        "video/mp4",
	"m4v":        "video/mp4",
	"mpg":        "video/mpeg",
	"mpeg":       "video/mpeg",
	"m2v":        "video/mpeg",
	"wmv":        "video/x-ms-wmv",
	"3g2":        "video/3gpp2",
	"3gp":        "video/3gpp",
	"asf":        "video/x-ms-asf",
	"avi":        "video/x-msvideo",
	"dat":        "video/mp2t",
	"m2ts":       "video/mp2t",
	"mts":        "video/mp2t",
	"dv":         "video/dv",
	"f4v":        "video/x-f4v",
	"flv":        "video/x-flv",
	"hevc":       "video/hevc",
	"mjpeg":      "video/mjpeg",
	"mkv":        "video/x-matroska",
	"mov":        "video/quicktime",
	"mxf":        "application/mxf",
	"ogv":        "video/ogg",
	"rm":         "application/vnd.rn-realmedia",
	"rmvb":       "application/vnd.rn-realmedia-vbr",
	"vob":        "video/x-ms-vob",
	"webm":       "video/webm",
	"wtv":        "video/x-ms-wtv",
	"mscompress": "application/x-mscompress",
	"hlp":        "application/winhlp",
	"md":         "text/markdown",
	"go":         "text/x-go",
	"java":       "text/x-java-source",
	"py":         "text/x-python",
	"c":          "text/x-c",
	"h":          "text/x-c",
	"cpp":        "text/x-c++",
	"hpp":        "text/x-c++",
	"php":        "application/x-httpd-php",
	"rb":         "text/x-ruby",
	"vbs":        "text/vbscript",
	"rs":         "text/x-rust",
	"swift":      "text/x-swift",
	"kt":         "text/x-kotlin",
	"sql":        "application/sql",
	"sh":         "application/x-sh",
	"bash":       "application/x-sh",
	"bat":        "application/x-bat",
	"ps1":        "application/x-powershell",
	"apk":        "application/vnd.android.package-archive",
	"azw3":       "application/vnd.amazon.ebook",
	"blend":      "application/x-blender",
	"c4d":        "application/vnd.c4d",
	"catpart":    "application/vnd.catia",
	"chm":        "application/vnd.ms-htmlhelp",
	"daf":        "application/x-daf",
	"dbf":        "application/x-dbf",
	"dcm":        "application/dicom",
	"djvu":       "image/vnd.djvu",
	"dsm":        "application/x-dsm",
	"dwg":        "application/acad",
	"dws":        "application/x-dws",
	"dxf":        "application/dxf",
	"eml":        "message/rfc822",
	"mht":        "message/rfc822",
	"mhtml":      "message/rfc822",
	"fbx":        "application/x-fbx",
	"in":         "text/plain",
	"jar":        "application/java-archive",
	"lrf":        "application/x-sony-bbeb",
	"m3u":        "audio/x-mpegurl",
	"m3u8":       "application/vnd.apple.mpegurl",
	"max":        "application/x-3ds",
	"prt":        "application/x-prt",
	"sldasm":     "application/vnd.solidworks",
	"sldprt":     "application/vnd.solidworks",
	"snb":        "application/x-snb",
	"stl":        "application/sla",
	"tex":        "application/x-tex",
	"vcf":        "text/vcard",
	"x3d":        "model/x3d+xml",
	"xpi":        "application/x-xpinstall",
	"xps":        "application/vnd.ms-xpsdocument",
	// 第 14 轮补充：DLP 测试数据集中的其他常见格式
	"3dm":     "model/vnd.3dm",
	"3mf":     "model/3mf",
	"cfg":     "text/x-cfg",
	"conf":    "text/x-conf",
	"config":  "text/x-config",
	"def":     "text/x-def",
	"dwf":     "model/vnd.dwf",
	"egg":     "application/x-python-egg",
	"f3d":     "application/vnd.autodesk.f3d",
	"iges":    "model/iges",
	"igs":     "model/iges",
	"key":     "application/vnd.apple.keynote",
	"list":    "text/x-list",
	"numbers": "application/vnd.apple.numbers",
	"obj":     "model/obj",
	"odp":     "application/vnd.oasis.opendocument.presentation",
	"ods":     "application/vnd.oasis.opendocument.spreadsheet",
	"ofd":     "application/vnd.ofd",
	"ott":     "application/vnd.oasis.opendocument.text-template",
	"pages":   "application/vnd.apple.pages",
	"pdb":     "application/vnd.palm",
	"rp":      "application/vnd.rn-realmedia",
	"step":    "model/step",
	"stp":     "model/step",
	"whl":     "application/x-python-wheel",
	"x_t":     "model/x-parasolid",
	"xmind":   "application/x-xmind",
	"zipx":    "application/zipx",
}

// MapExtensionToMimeType 将文件扩展名映射到MIME类型（O(1) map 查表）。
func MapExtensionToMimeType(ext string) string {
	if m, ok := extToMimeMap[ext]; ok {
		return m
	}
	return "application/octet-stream"
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
	_, ok := officeExtensionsMap[fileType]
	return ok
}

// IsArchiveFile 判断是否是压缩文件
func (d *FileTypeDetector) IsArchiveFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}
	_, ok := archiveExtensionsMap[fileType]
	return ok
}

// IsImageFile 判断是否是图片文件
func (d *FileTypeDetector) IsImageFile(filePath string) bool {
	fileType, err := d.DetectFileType(filePath)
	if err != nil {
		return false
	}
	_, ok := imageExtensionsMap[fileType]
	return ok
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
