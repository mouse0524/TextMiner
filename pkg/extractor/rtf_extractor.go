package extractor

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
	ctx, err := prepareExtractContext(filePath)
	if err != nil {
		return newFileAccessErrorResult(filePath), fmt.Errorf("文件不存在或无法访问")
	}
	result := newSuccessResult(ctx, "")

	f, err := os.Open(filePath)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = fmt.Sprintf("读取文件失败: %v", err)
		return result, err
	}
	defer f.Close()

	br := getBufioReader(f)
	defer putBufioReader(br)
	data, err := io.ReadAll(br)
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

// codePageDecoders O(1) 查表：替换 13 路 switch；找不到时返回 nil（调用方 fallback）。
var codePageDecoders = map[int]encoding.Encoding{
	936:  simplifiedchinese.GBK,
	950:  traditionalchinese.Big5,
	1250: charmap.Windows1250,
	1251: charmap.Windows1251,
	1252: charmap.Windows1252,
	1253: charmap.Windows1253,
	1254: charmap.Windows1254,
	1255: charmap.Windows1255,
	1256: charmap.Windows1256,
	1257: charmap.Windows1257,
	1258: charmap.Windows1258,
}

func getDecoder(codePage int) encoding.Encoding {
	return codePageDecoders[codePage]
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

// symbolReplacements O(1) 查表：替换 14 路 switch；找不到时返回 "" + false。
var symbolReplacements = map[string]string{
	"bullet":    "*",
	"emdash":    "-",
	"endash":    "-",
	"lquote":    "'",
	"rquote":    "'",
	"ldblquote": "\"",
	"rdblquote": "\"",
	"line":      "\n",
	"par":       "\n",
	"page":      "\n",
	"sect":      "\n",
	"tab":       "\t",
	"cell":      " ",
	"column":    " ",
	"row":       " ",
	"~":         " ",
	"_":         "-",
	"|":         "|",
	"-":         "-",
	":":         ":",
	"{":         "",
	"}":         "",
}

func convertSymbol(symbol string) (string, bool) {
	if v, ok := symbolReplacements[symbol]; ok {
		return v, true
	}
	return "", false
}
