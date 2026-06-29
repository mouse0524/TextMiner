package magika

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
)

var (
	scanner     *Scanner
	scannerOnce sync.Once
)

// InitScanner initializes the Magika scanner with the given assets directory.
// It should be called once before using DetectFileType.
func InitScanner(assetsDir string) error {
	var initErr error
	scannerOnce.Do(func() {
		s, err := NewScanner(assetsDir, "")
		if err != nil {
			initErr = err
			return
		}
		scanner = s
	})
	return initErr
}

// DetectFileType detects the file type of the given file using Magika.
// It returns the file type label and an error if detection fails.
func DetectFileType(filePath string) (string, error) {
	if scanner == nil {
		return "", ErrScannerNotInitialized
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	ct, err := scanner.Scan(file, int(info.Size()))
	if err != nil {
		return "", err
	}

	return mapContentTypeToFileType(ct), nil
}

// DetectFileTypeFromBytes detects the file type from byte data using Magika.
func DetectFileTypeFromBytes(data []byte) (string, error) {
	if scanner == nil {
		return "", ErrScannerNotInitialized
	}

	if len(data) == 0 {
		return "empty", nil
	}

	ct, err := scanner.Scan(bytes.NewReader(data), len(data))
	if err != nil {
		return "", err
	}

	return mapContentTypeToFileType(ct), nil
}

// mapContentTypeToFileType maps Magika ContentType to a simple file type string.
func mapContentTypeToFileType(ct ContentType) string {
	switch ct.Label {
	case "doc":
		return "doc"
	case "docx":
		return "docx"
	case "xls":
		return "xls"
	case "xlsx":
		return "xlsx"
	case "ppt":
		return "ppt"
	case "pptx":
		return "pptx"
	case "pdf":
		return "pdf"
	case "txt":
		return "txt"
	case "zip":
		return "zip"
	case "7z":
		return "7z"
	case "rar":
		return "rar"
	case "tar":
		return "tar"
	case "gz":
		return "gz"
	case "png":
		return "png"
	case "jpg", "jpeg":
		return "jpg"
	case "bmp":
		return "bmp"
	case "gif":
		return "gif"
	case "webp":
		return "webp"
	case "html":
		return "html"
	case "xml":
		return "xml"
	case "json":
		return "json"
	case "textproto":
		return "txt"
	case "empty":
		return "empty"
	case "unknown":
		return "unknown"
	case "mid", "midi":
		return "mid"
	case "wav":
		return "wav"
	case "ogg":
		return "ogg"
	case "mp3":
		return "mp3"
	case "8svx":
		return "8svx"
	case "aac":
		return "aac"
	case "ac3":
		return "ac3"
	case "aiff":
		return "aiff"
	case "amb":
		return "amb"
	case "amr":
		return "amr"
	case "au":
		return "au"
	case "avr":
		return "avr"
	case "caf":
		return "caf"
	case "cdda":
		return "cdda"
	case "cvs":
		return "cvs"
	case "cvu":
		return "cvu"
	case "dts":
		return "dts"
	case "dvms":
		return "dvms"
	case "fap":
		return "fap"
	case "flac":
		return "flac"
	case "fssd":
		return "fssd"
	case "gsrt":
		return "gsrt"
	case "hcom":
		return "hcom"
	case "htk":
		return "htk"
	case "ima":
		return "ima"
	case "ircam":
		return "ircam"
	case "m4a":
		return "m4a"
	case "m4r":
		return "m4r"
	case "maud":
		return "maud"
	case "mmf":
		return "mmf"
	case "mp2":
		return "mp2"
	case "nist":
		return "nist"
	case "oga":
		return "oga"
	case "opus":
		return "opus"
	case "paf":
		return "paf"
	case "pcma":
		return "pcma"
	case "pcmu":
		return "pcmu"
	case "prc":
		return "prc"
	case "pvf":
		return "pvf"
	case "ra":
		return "ra"
	case "sd2":
		return "sd2"
	case "sln":
		return "sln"
	case "smp":
		return "smp"
	case "snd":
		return "snd"
	case "sndr":
		return "sndr"
	case "sndt":
		return "sndt"
	case "sou":
		return "sou"
	case "sph":
		return "sph"
	case "spx":
		return "spx"
	case "tta":
		return "tta"
	case "txw":
		return "txw"
	case "vms":
		return "vms"
	case "voc":
		return "voc"
	case "vox":
		return "vox"
	case "w64":
		return "w64"
	case "wma":
		return "wma"
	case "wv":
		return "wv"
	case "wve":
		return "wve"
	default:
		ext := ""
		if len(ct.Extensions) > 0 {
			ext = ct.Extensions[0]
		}
		if ext != "" {
			return ext
		}
		return ct.Label
	}
}

// MapContentTypeToFileType maps Magika ContentType to a simple file type string (exported).
func MapContentTypeToFileType(ct ContentType) string {
	return mapContentTypeToFileType(ct)
}

// GetContentType returns the full ContentType information for a file.
func GetContentType(filePath string) (ContentType, error) {
	if scanner == nil {
		return ContentType{}, ErrScannerNotInitialized
	}

	file, err := os.Open(filePath)
	if err != nil {
		return ContentType{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ContentType{}, err
	}

	return scanner.Scan(file, int(info.Size()))
}

// IsInitialized returns true if the Magika scanner has been initialized.
func IsInitialized() bool {
	return scanner != nil
}

// GetDefaultAssetsDir returns the default assets directory path.
func GetDefaultAssetsDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(execPath), "models"), nil
}
