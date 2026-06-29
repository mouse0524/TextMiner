package goppt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/richardlehane/mscfb"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/KSpaceer/goppt/internal/ioadapters"
)

type mmapReader struct {
	data     []byte
	file     *os.File
	fileSize int64
}

func newMmapReader(file *os.File) (*mmapReader, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil, fmt.Errorf("empty file")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &mmapReader{
		data:     data,
		file:     file,
		fileSize: fileSize,
	}, nil
}

func (m *mmapReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= m.fileSize {
		return 0, io.EOF
	}
	if off+int64(len(p)) > m.fileSize {
		n = int(m.fileSize - off)
		copy(p, m.data[off:off+int64(n)])
		return n, io.EOF
	}
	copy(p, m.data[off:off+int64(len(p))])
	return len(p), nil
}

func (m *mmapReader) Close() error {
	m.data = nil
	m.fileSize = 0
	return nil
}

// skipped metadata or non-readable records in slide container
var slideSkippedRecordsTypes = []recordType{
	recordTypeExternalObjectList,
	recordTypeEnvironment,
	recordTypeSoundCollection,
	recordTypeDrawingGroup,
	recordTypeSlideListWithText,
	recordTypeList,
	recordTypeHeadersFooters,
	recordTypeHeadersFooters,
}

// skipped metadata or non-readable records in drawing container
var drawingSkippedRecordsTypes = []recordType{
	recordTypeSlideShowSlideInfoAtom,
	recordTypeHeadersFooters,
	recordTypeRoundTripSlideSyncInfo12,
}

const (
	userPersistIDRefOffset = 16
)

// ExtractText parses PPT file represented by Reader r and extracts text from it.
func ExtractText(r io.Reader) (string, error) {
	ra := ioadapters.ToReaderAt(r)

	d, err := mscfb.New(ra)
	if err != nil {
		return "", err
	}
	currentUser, pptDocument := getCurrentUserAndPPTDoc(d)
	if err := isValidPPT(currentUser, pptDocument); err != nil {
		return "", err
	}
	offsetPersistDirectory, liveRecord, err := getUserEditAtomsData(currentUser, pptDocument)
	if err != nil {
		return "", err
	}
	persistDirEntries, err := getPersistDirectoryEntries(pptDocument, offsetPersistDirectory)
	if err != nil {
		return "", err
	}

	// get DocumentContainer reference
	docPersistIDRef := liveRecord.LongAt(userPersistIDRefOffset)
	documentContainer, err := readRecord(pptDocument, persistDirEntries[docPersistIDRef], recordTypeDocument)
	if err != nil {
		return "", err
	}

	return readSlides(documentContainer, pptDocument, persistDirEntries)
}

// ExtractTextWithMmap parses PPT file using memory-mapped I/O for better performance
func ExtractTextWithMmap(file *os.File) (string, error) {
	mmap, err := newMmapReader(file)
	if err != nil {
		return "", err
	}
	defer mmap.Close()

	d, err := mscfb.New(mmap)
	if err != nil {
		return "", err
	}
	currentUser, pptDocument := getCurrentUserAndPPTDoc(d)
	if err := isValidPPT(currentUser, pptDocument); err != nil {
		return "", err
	}
	offsetPersistDirectory, liveRecord, err := getUserEditAtomsData(currentUser, pptDocument)
	if err != nil {
		return "", err
	}
	persistDirEntries, err := getPersistDirectoryEntries(pptDocument, offsetPersistDirectory)
	if err != nil {
		return "", err
	}

	// get DocumentContainer reference
	docPersistIDRef := liveRecord.LongAt(userPersistIDRefOffset)
	documentContainer, err := readRecord(pptDocument, persistDirEntries[docPersistIDRef], recordTypeDocument)
	if err != nil {
		return "", err
	}

	return readSlides(documentContainer, pptDocument, persistDirEntries)
}

// toMemoryBuffer transforms io.Reader to in-memory io.ReaderAt

// getCurrentUserAndPPTDoc extracts necessary mscfb files from PPT file
func getCurrentUserAndPPTDoc(r *mscfb.Reader) (currentUser *mscfb.File, pptDocument *mscfb.File) {
	for _, f := range r.File {
		switch f.Name {
		case "Current User":
			currentUser = f
		case "PowerPoint Document":
			pptDocument = f
		}
	}
	return currentUser, pptDocument
}

// isValidPPT checks if provided file is valid, meaning
// it has both "Current User" and "PowerPoint Document" files
// and "Current User"'s CurrentUserAtom record has valid header token
func isValidPPT(currentUser, pptDocument *mscfb.File) error {
	const (
		headerTokenOffset      = 12
		encryptedDocumentToken = 0xF3D1C4DF
		plainDocumentToken     = 0xE391C05F
	)

	if currentUser == nil || pptDocument == nil {
		return fmt.Errorf(".ppt file must contain \"Current User\" and \"PowerPoint Document\" streams")
	}
	var b [4]byte
	_, err := currentUser.ReadAt(b[:], headerTokenOffset)
	if err != nil {
		return err
	}
	headerToken := binary.LittleEndian.Uint32(b[:])
	if headerToken != plainDocumentToken && headerToken != encryptedDocumentToken {
		return fmt.Errorf("invalid UserEditAtom header token %X", headerToken)
	}
	return nil
}

// getUserEditAtomsData extracts "live record" and persist directory offsets
// according to section 2.1.2 of specification (https://msopenspecs.azureedge.net/files/MS-PPT/%5bMS-PPT%5d-210422.pdf)
func getUserEditAtomsData(currentUser, pptDocument *mscfb.File) (
	persistDirectoryOffsets []int64,
	liveRecord record,
	err error,
) {
	const (
		offsetLastEditInitialPosition  = 16
		offsetLastEditPosition         = 8
		persistDirectoryOffsetPosition = 12
	)
	var b [4]byte
	_, err = currentUser.ReadAt(b[:], offsetLastEditInitialPosition)
	if err != nil {
		return nil, record{}, err
	}
	offsetLastEdit := binary.LittleEndian.Uint32(b[:])

	for {
		liveRecord, err = readRecord(pptDocument, int64(offsetLastEdit), recordTypeUserEditAtom)
		if err != nil {
			if errors.Is(err, errMismatchRecordType) {
				break
			}
			return nil, record{}, err
		}
		persistDirectoryOffsets = append(
			persistDirectoryOffsets,
			int64(liveRecord.LongAt(persistDirectoryOffsetPosition)),
		)
		offsetLastEdit = liveRecord.LongAt(offsetLastEditPosition)
		if offsetLastEdit == 0 {
			break
		}
	}

	return persistDirectoryOffsets, liveRecord, err
}

// getPersistDirectoryEntries transforms offsets into persists directory identifiers and persist offsets according
// to section 2.1.2 of specification (https://msopenspecs.azureedge.net/files/MS-PPT/%5bMS-PPT%5d-210422.pdf)
func getPersistDirectoryEntries(pptDocument *mscfb.File, offsets []int64) (map[uint32]int64, error) {
	const persistOffsetEntrySize = 4

	persistDirEntries := make(map[uint32]int64)
	for i := len(offsets) - 1; i >= 0; i-- {
		rgPersistDirEntry, err := readRecord(pptDocument, offsets[i], recordTypePersistDirectoryAtom)
		if err != nil {
			if errors.Is(err, errMismatchRecordType) {
				continue
			}
			if errors.Is(err, io.EOF) {
				continue
			}
			return nil, err
		}

		rgPersistDirEntryData := rgPersistDirEntry.recordData

		for j := 0; j < len(rgPersistDirEntryData); {
			if j+4 > len(rgPersistDirEntryData) {
				break
			}
			persist := rgPersistDirEntryData.LongAt(j)
			persistID := persist & 0x000FFFFF
			cPersist := ((persist & 0xFFF00000) >> 20) & 0x00000FFF
			j += 4

			for k := uint32(0); k < cPersist; k++ {
				offset := j + int(k)*persistOffsetEntrySize
				if offset+4 > len(rgPersistDirEntryData) {
					break
				}
				persistDirEntries[persistID+k] = int64(rgPersistDirEntryData.LongAt(offset))
			}
			j += int(cPersist * persistOffsetEntrySize)
		}
	}
	return persistDirEntries, nil
}

// readSlides reads text from slides of given DocumentContainer
func readSlides(documentContainer, pptDocument io.ReaderAt, persistDirEntries map[uint32]int64) (string, error) {
	const slideSkipInitialOffset = 48
	offset, err := skipRecords(documentContainer, slideSkipInitialOffset, slideSkippedRecordsTypes)
	if err != nil {
		return "", err
	}
	slideList, err := readRecord(documentContainer, offset, recordTypeSlideListWithText)
	if err != nil {
		return "", err
	}

	utf16Decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()

	slideListData := slideList.Data()
	estimatedSize := len(slideListData)
	if estimatedSize < 1024*1024 {
		estimatedSize = 1024 * 1024
	}
	var out strings.Builder
	out.Grow(estimatedSize)

	n := len(slideListData)
	for i := 0; i < n; {
		block, err := readRecord(slideList, int64(i), recordTypeUnspecified)
		if err != nil {
			return "", err
		}
		blockType := block.Type()
		blockData := block.Data()

		switch blockType {
		case recordTypeSlidePersistAtom:
			err = readTextFromSlidePersistAtom(block, pptDocument, persistDirEntries, &out, utf16Decoder)
		case recordTypeTextCharsAtom:
			err = readTextFromTextCharsAtom(block, &out, utf16Decoder)
		case recordTypeTextBytesAtom:
			err = readTextFromTextBytesAtom(block, &out, utf16Decoder)
		}
		if err != nil {
			return "", err
		}

		i += len(blockData) + 8
	}

	return out.String(), nil
}

func readTextFromSlidePersistAtom(
	block record,
	pptDocument io.ReaderAt,
	persistDirEntries map[uint32]int64,
	out *strings.Builder,
	utf16Decoder *encoding.Decoder,
) error {
	const (
		slidePersistAtomSkipInitialOffset = 32
		headerRecordTypeOffset            = 2
	)

	persistDirID := block.LongAt(0)
	// extract slide from persist directory
	slide, err := readRecord(pptDocument, persistDirEntries[persistDirID], recordTypeSlide)
	if err != nil {
		return err
	}
	// skip metadata
	offset, err := skipRecords(slide, slidePersistAtomSkipInitialOffset, drawingSkippedRecordsTypes)
	if err != nil {
		return err
	}

	drawing, err := readRecord(slide, offset, recordTypeDrawing)
	if err != nil {
		return err
	}
	drawingBytes := drawing.Data()
	from := 0

	textRecordCount := 0
	for {
		pocketIdx := matchPocket(drawingBytes, from)
		if pocketIdx == -1 {
			break
		}
		if pocketIdx >= 2 && bytes.Equal(drawingBytes[pocketIdx-headerRecordTypeOffset:pocketIdx], []byte{0x00, 0x00}) {
			var rec record
			var err error
			if drawingBytes[pocketIdx] == recordTypeTextBytesAtom.LowerPart() {
				rec, err = readRecord(drawing, int64(pocketIdx-headerRecordTypeOffset), recordTypeTextBytesAtom)
				if err == nil {
					err = readTextFromTextBytesAtom(rec, out, utf16Decoder)
				}
			} else {
				rec, err = readRecord(drawing, int64(pocketIdx-headerRecordTypeOffset), recordTypeTextCharsAtom)
				if err == nil {
					err = readTextFromTextCharsAtom(rec, out, utf16Decoder)
				}
			}
			if err != nil {
				return err
			}
			textRecordCount++
		}
		from = pocketIdx + 2
	}

	return nil
}

func matchPocket(data []byte, from int) int {
	data = data[from:]
	n := len(data)
	for i := 0; i < n-1; i++ {
		b := data[i]
		if (b == recordTypeTextCharsAtom.LowerPart() || b == recordTypeTextBytesAtom.LowerPart()) && data[i+1] == 0x0F {
			return i + from
		}
	}
	return -1
}

// readTextFromTextCharsAtom simply transforms UTF-16LE data into UTF-8 data
func readTextFromTextCharsAtom(atom record, out *strings.Builder, dec *encoding.Decoder) error {
	data := atom.Data()
	if len(data) == 0 {
		return nil
	}

	dec.Reset()
	transformed, err := dec.Bytes(data)
	if err != nil {
		return err
	}
	out.Write(transformed)
	out.WriteByte(' ')
	return nil
}

// readTextFromTextBytesAtom transforms text from TextBytesAtom into UTF-8 data
func readTextFromTextBytesAtom(atom record, out *strings.Builder, dec *encoding.Decoder) error {
	data := atom.Data()
	if len(data) == 0 {
		return nil
	}

	dec.Reset()
	transformed, err := decodeTextBytesAtom(data, dec)
	if err != nil {
		return err
	}
	out.Write(transformed)
	out.WriteByte(' ')
	return nil
}

// decodeTextBytesAtom transforms text from TextBytesAtom, which is an array of bytes representing lower parts of UTF-16
// characters into UTF-8 data
func decodeTextBytesAtom(data []byte, dec *encoding.Decoder) ([]byte, error) {
	var (
		buf [2]byte
		err error
	)
	result := make([]byte, 0, len(data)*2)
	for i := range data {
		buf[0] = data[i]
		buf[1] = 0

		result, _, err = transform.Append(dec, result, buf[:])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// skipRecords reads headers and skips data of records of provided types
func skipRecords(r io.ReaderAt, initialOffset int64, skippedRecordsTypes []recordType) (int64, error) {
	offset := initialOffset

	for i := range skippedRecordsTypes {
		rec, err := readRecordHeaderOnly(r, offset, skippedRecordsTypes[i])
		if err != nil {
			if errors.Is(err, errMismatchRecordType) {
				continue
			}
			return 0, err
		}
		offset += int64(rec.Length() + headerSize)
	}

	return offset, nil
}
