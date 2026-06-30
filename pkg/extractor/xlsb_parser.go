package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	BrtRowHdr         = 0
	BrtCellBlank      = 1
	BrtCellRk         = 2
	BrtCellError      = 3
	BrtCellBool       = 4
	BrtCellReal       = 5
	BrtCellSt         = 6
	BrtCellIsst       = 7
	BrtSSTItem        = 19
	BrtBeginSheet     = 129
	BrtEndSheet       = 130
	BrtBeginBook      = 131
	BrtEndBook        = 132
	BrtBeginSst       = 159
	BrtEndSst         = 160
	BrtBeginSheetData = 145
	BrtEndSheetData   = 146
)

type Record struct {
	ID   uint16
	Data []byte
}

type XlsbParser struct {
	sharedStrings []string
}

func NewXlsbParser() *XlsbParser {
	return &XlsbParser{
		sharedStrings: make([]string, 0),
	}
}

func (p *XlsbParser) Parse(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开XLSB文件失败（仅支持ZIP容器格式的新版XLSB）: %w", err)
	}
	defer r.Close()

	var result strings.Builder
	result.Grow(1024 * 1024)

	for _, file := range r.File {
		if file.Name == "xl/sharedStrings.bin" {
			f, err := file.Open()
			if err != nil {
				continue
			}

			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}

			p.parseSharedStrings(data)
		} else if strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".bin") {
			f, err := file.Open()
			if err != nil {
				continue
			}

			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}

			text := p.parseWorksheet(data)
			if text != "" {
				result.WriteString(text)
				result.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(result.String()), nil
}

func (p *XlsbParser) parseWorksheet(data []byte) string {
	var result strings.Builder
	result.Grow(len(data) / 8)

	pos := 0
	dataLen := len(data)

	for pos < dataLen-8 {
		recordID, recordLen, bytesRead := p.readRecordFast(data[pos:])
		if recordID == 0 && recordLen == 0 {
			break
		}

		pos += bytesRead

		if pos+int(recordLen) > dataLen {
			break
		}

		recordData := data[pos : pos+int(recordLen)]
		pos += int(recordLen)

		switch recordID {
		case BrtCellSt:
			if text := p.parseCellStFast(recordData); text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		case BrtCellIsst:
			if text := p.parseCellIsstFast(recordData); text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		case BrtCellRk:
			if text := p.parseCellRkFast(recordData); text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		case BrtCellReal:
			if text := p.parseCellRealFast(recordData); text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		case BrtCellBool:
			if text := p.parseCellBoolFast(recordData); text != "" {
				result.WriteString(text)
				result.WriteString(" ")
			}
		}
	}

	return strings.TrimSpace(result.String())
}

func (p *XlsbParser) readRecordFast(data []byte) (uint16, uint32, int) {
	if len(data) < 2 {
		return 0, 0, 0
	}

	recordID, bytesRead := p.readVarIntFast(data, 2)
	dataLen, bytesRead2 := p.readVarIntFast(data[bytesRead:], 4)

	return uint16(recordID), dataLen, bytesRead + bytesRead2
}

func (p *XlsbParser) readVarIntFast(data []byte, maxBytes int) (uint32, int) {
	var result uint32
	var bytesRead int

	for bytesRead < maxBytes && bytesRead < len(data) {
		b := data[bytesRead]
		result |= (uint32(b&0x7f) << (7 * bytesRead))
		bytesRead++

		if (b & 0x80) == 0 {
			break
		}
	}

	return result, bytesRead
}

func (p *XlsbParser) readVarInt(r *bytes.Reader, maxBytes int) (uint32, error) {
	var result uint32
	var bytesRead int

	for bytesRead < maxBytes {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= (uint32(b&0x7f) << (7 * bytesRead))
		bytesRead++

		if (b & 0x80) == 0 {
			break
		}
	}

	return result, nil
}

func (p *XlsbParser) parseSharedStrings(data []byte) {
	pos := 0
	dataLen := len(data)

	p.sharedStrings = make([]string, 0)

	for pos < dataLen-8 {
		recordID, recordLen, bytesRead := p.readRecordFast(data[pos:])
		if recordID == 0 && recordLen == 0 {
			break
		}

		pos += bytesRead

		if pos+int(recordLen) > dataLen {
			break
		}

		recordData := data[pos : pos+int(recordLen)]
		pos += int(recordLen)

		if recordID == BrtSSTItem {
			if str := p.parseSSTItem(recordData); str != "" {
				p.sharedStrings = append(p.sharedStrings, str)
			}
		}
	}
}

func (p *XlsbParser) parseSSTItem(data []byte) string {
	if len(data) < 5 {
		return ""
	}

	pos := 1

	if pos+4 > len(data) {
		return ""
	}

	length := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4

	if length == 0 || length > 65536 || pos+int(length*2) > len(data) {
		return ""
	}

	utf16Data := make([]uint16, length)
	for j := uint32(0); j < length; j++ {
		utf16Data[j] = binary.LittleEndian.Uint16(data[pos+int(j*2) : pos+int(j*2)+2])
	}

	return utf16leToString(utf16Data)
}

func utf16leToString(utf16 []uint16) string {
	if len(utf16) == 0 {
		return ""
	}

	buf := make([]byte, 4*len(utf16))
	n := 0
	i := 0

	for i < len(utf16) {
		r := utf16[i]
		i++

		if r == 0 {
			continue
		}

		if r < 0x80 {
			buf[n] = byte(r)
			n++
		} else if r < 0x800 {
			buf[n] = byte(0xC0 | (r >> 6))
			buf[n+1] = byte(0x80 | (r & 0x3F))
			n += 2
		} else if r >= 0xD800 && r < 0xDC00 {
			if i >= len(utf16) {
				continue
			}
			r2 := utf16[i]
			i++
			if r2 < 0xDC00 || r2 >= 0xE000 {
				continue
			}
			runeValue := 0x10000 + (uint32(r-0xD800) << 10) + uint32(r2-0xDC00)
			buf[n] = byte(0xF0 | (runeValue >> 18))
			buf[n+1] = byte(0x80 | ((runeValue >> 12) & 0x3F))
			buf[n+2] = byte(0x80 | ((runeValue >> 6) & 0x3F))
			buf[n+3] = byte(0x80 | (runeValue & 0x3F))
			n += 4
		} else if r >= 0xDC00 && r < 0xE000 {
			continue
		} else {
			buf[n] = byte(0xE0 | (r >> 12))
			buf[n+1] = byte(0x80 | ((r >> 6) & 0x3F))
			buf[n+2] = byte(0x80 | (r & 0x3F))
			n += 3
		}
	}

	return string(buf[:n])
}

func (p *XlsbParser) readWideString(r *bytes.Reader) (string, error) {
	var length uint32
	err := binary.Read(r, binary.LittleEndian, &length)
	if err != nil {
		return "", err
	}

	if length == 0 || length > 65536 {
		return "", nil
	}

	strBytes := make([]byte, length*2)
	_, err = io.ReadFull(r, strBytes)
	if err != nil {
		return "", err
	}

	result := make([]rune, 0, length)
	for i := uint32(0); i < length; i++ {
		char := binary.LittleEndian.Uint16(strBytes[i*2 : i*2+2])
		result = append(result, rune(char))
	}

	return string(result), nil
}

func (p *XlsbParser) parseCellStFast(data []byte) string {
	if len(data) < 8 {
		return ""
	}

	pos := 8

	if pos+4 > len(data) {
		return ""
	}

	length := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4

	if length == 0 || length > 65536 || pos+int(length*2) > len(data) {
		return ""
	}

	utf16Data := make([]uint16, length)
	for j := uint32(0); j < length; j++ {
		utf16Data[j] = binary.LittleEndian.Uint16(data[pos+int(j*2) : pos+int(j*2)+2])
	}

	return utf16leToString(utf16Data)
}

func (p *XlsbParser) parseCellIsstFast(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	sstIndex := binary.LittleEndian.Uint32(data[8:12])

	if int(sstIndex) < len(p.sharedStrings) {
		return p.sharedStrings[sstIndex]
	}

	return ""
}

// xlsbCellBufferPool 复用 *bytes.Buffer 减少单元格格式化时 GC 压力。
// strconv.Append* 写入 buffer；String() 后归还 pool。注意：调用方在 String() 后
// 不再持有 buffer 引用，因此 pool 重用是安全的。
var xlsbCellBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 32))
	},
}

func (p *XlsbParser) parseCellRkFast(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	value := binary.LittleEndian.Uint64(data[8:16])

	buf := xlsbCellBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer xlsbCellBufferPool.Put(buf)

	if value&0x02 != 0 {
		floatVal := float64(int32(value>>2)) / 100.0
		if floatVal == float64(int64(floatVal)) {
			buf.Write(strconv.AppendInt(buf.Bytes()[:0], int64(floatVal), 10))
		} else {
			buf.Write(strconv.AppendFloat(buf.Bytes()[:0], floatVal, 'g', -1, 64))
		}
	} else {
		buf.Write(strconv.AppendInt(buf.Bytes()[:0], int64(int32(value>>2)), 10))
	}
	return buf.String()
}

func (p *XlsbParser) parseCellRealFast(data []byte) string {
	if len(data) < 16 {
		return ""
	}

	value := binary.LittleEndian.Uint64(data[8:16])
	floatValue := math.Float64frombits(value)

	buf := xlsbCellBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer xlsbCellBufferPool.Put(buf)

	if floatValue == float64(int64(floatValue)) {
		buf.Write(strconv.AppendInt(buf.Bytes()[:0], int64(floatValue), 10))
	} else {
		buf.Write(strconv.AppendFloat(buf.Bytes()[:0], floatValue, 'g', -1, 64))
	}
	return buf.String()
}

func (p *XlsbParser) parseCellBoolFast(data []byte) string {
	if len(data) < 9 {
		return ""
	}

	value := data[8]

	if value != 0 {
		return "TRUE"
	}
	return "FALSE"
}

func (p *XlsbParser) readWideStringFast(data []byte) (string, error) {
	if len(data) < 4 {
		return "", nil
	}

	length := binary.LittleEndian.Uint32(data[0:4])

	if length == 0 || length > 65536 {
		return "", nil
	}

	if len(data) < int(4+length*2) {
		return "", nil
	}

	result := make([]rune, 0, length)
	for i := uint32(0); i < length; i++ {
		char := binary.LittleEndian.Uint16(data[4+i*2 : 4+i*2+2])
		result = append(result, rune(char))
	}

	return string(result), nil
}
