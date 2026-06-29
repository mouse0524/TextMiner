package extractor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unicode/utf8"

	"github.com/EndFirstCorp/peekingReader"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

type RtfExtractor struct{}

func (e *RtfExtractor) Extract(filePath string, enableOcr bool) (*ExtractResult, error) {
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}

	content, err := extractRtfText(data)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("提取RTF内容失败: %v", err)
		return result, err
	}

	result.Content = content
	return result, nil
}

func extractRtfText(data []byte) (string, error) {
	reader := bytes.NewReader(data)
	pr := peekingReader.NewBufReader(reader)

	var text bytes.Buffer
	var decoder encoding.Encoding = simplifiedchinese.GBK
	var hexBuffer bytes.Buffer

	for b, err := pr.ReadByte(); err == nil; b, err = pr.ReadByte() {
		switch b {
		case '\\':
			err := readControl(pr, &text, &decoder, &hexBuffer)
			if err != nil {
				return "", err
			}
		case '{', '}':
		case '\n', '\r':
		default:
			hexBuffer.WriteByte(b)
		}
	}

	if hexBuffer.Len() > 0 {
		decoded, err := decodeBytes(hexBuffer.Bytes(), decoder)
		if err == nil {
			text.WriteString(decoded)
		}
	}

	result := text.String()

	if !utf8.ValidString(result) {
		return "", fmt.Errorf("RTF内容包含无效的UTF-8字符")
	}

	return result, nil
}

func decodeBytes(data []byte, decoder encoding.Encoding) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), decoder.NewDecoder())
	result, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func readControl(r peekingReader.Reader, text *bytes.Buffer, decoder *encoding.Encoding, hexBuffer *bytes.Buffer) error {
	control, num, err := tokenizeControl(r)
	if err != nil {
		return err
	}

	if control == "*" {
		return readUntilClosingBrace(r)
	}

	if isHexByte, b := getHexByte(control); isHexByte {
		hexBuffer.WriteByte(b)
		return nil
	}

	if control == "" {
		p, err := r.Peek(1)
		if err != nil {
			return err
		}
		if p[0] == '\\' || p[0] == '{' || p[0] == '}' {
			text.WriteByte(p[0])
			r.ReadByte()
			return nil
		}
		text.WriteByte('\n')
		return nil
	}

	if control == "u" {
		if num < 0 {
			num += 65536
		}
		text.WriteRune(rune(num))
		return nil
	}

	if control == "ansicpg" {
		*decoder = getDecoder(num)
		return nil
	}

	if control == "fonttbl" {
		return readUntilClosingBrace(r)
	}

	if control == "colortbl" {
		return readUntilClosingBrace(r)
	}

	if control == "stylesheet" {
		return readUntilClosingBrace(r)
	}

	if control == "info" {
		return readUntilClosingBrace(r)
	}

	if symbol, found := convertSymbol(control); found {
		text.WriteString(symbol)
	}

	return nil
}

func getDecoder(codePage int) encoding.Encoding {
	switch codePage {
	case 936:
		return simplifiedchinese.GBK
	case 950:
		return traditionalchinese.Big5
	case 1250:
		return charmap.Windows1250
	case 1251:
		return charmap.Windows1251
	case 1252:
		return charmap.Windows1252
	case 1253:
		return charmap.Windows1253
	case 1254:
		return charmap.Windows1254
	case 1255:
		return charmap.Windows1255
	case 1256:
		return charmap.Windows1256
	case 1257:
		return charmap.Windows1257
	case 1258:
		return charmap.Windows1258
	default:
		return nil
	}
}

func tokenizeControl(r peekingReader.Reader) (string, int, error) {
	var buf bytes.Buffer
	isHex := false
	numStart := -1
	for {
		p, err := r.Peek(1)
		if err != nil {
			return "", -1, err
		}
		b := p[0]
		switch {
		case b == '*' && buf.Len() == 0:
			r.ReadByte()
			return "*", -1, nil
		case b == '\'' && buf.Len() == 0:
			isHex = true
			buf.WriteByte(b)
			r.ReadByte()
			for i := 0; i < 2; i++ {
				b, err = r.ReadByte()
				if err != nil {
					return "", -1, err
				}
				buf.WriteByte(b)
			}
			return buf.String(), -1, nil
		case b >= '0' && b <= '9' || b == '-':
			if numStart == -1 {
				numStart = buf.Len()
			} else if numStart == 0 {
				return "", -1, fmt.Errorf("Unexpected control sequence. Cannot begin with digit")
			}
			buf.WriteByte(b)
			r.ReadByte()
		case b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z':
			if numStart > 0 {
				c, num := canonicalize(buf.String(), numStart)
				return c, num, nil
			}
			buf.WriteByte(b)
			r.ReadByte()
		default:
			if isHex {
				return buf.String(), -1, nil
			}
			c, num := canonicalize(buf.String(), numStart)
			return c, num, nil
		}
	}
}

func canonicalize(control string, numStart int) (string, int) {
	if numStart == -1 || numStart >= len(control) {
		return control, -1
	}
	num, err := strconv.Atoi(control[numStart:])
	if err != nil {
		return control, -1
	}
	return control[:numStart] + "N", num
}

func getHexByte(control string) (bool, byte) {
	if len(control) < 3 || control[0] != '\'' {
		return false, 0
	}

	hexStr := control[1:3]
	if len(hexStr) != 2 {
		return false, 0
	}

	num, err := strconv.ParseInt(hexStr, 16, 16)
	if err != nil {
		return false, 0
	}

	return true, byte(num)
}

func readUntilClosingBrace(r peekingReader.Reader) error {
	count := 1
	var b byte
	var err error
	for b, err = r.ReadByte(); err == nil; b, err = r.ReadByte() {
		switch b {
		case '{':
			count++
		case '}':
			count--
		}
		if count == 0 {
			return nil
		}
	}
	return err
}

func convertSymbol(symbol string) (string, bool) {
	switch symbol {
	case "bullet":
		return "*", true
	case "emdash", "endash":
		return "-", true
	case "lquote", "rquote":
		return "'", true
	case "ldblquote", "rdblquote":
		return "\"", true
	case "line":
		return "\n", true
	case "par", "page", "sect":
		return "\n", true
	case "tab":
		return "\t", true
	case "cell", "column", "row":
		return " ", true
	case "~":
		return " ", true
	case "_":
		return "-", true
	case "|":
		return "|", true
	case "-":
		return "-", true
	case ":":
		return ":", true
	case "{", "}":
		return "", true
	default:
		return "", false
	}
}
