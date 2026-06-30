package extractor

import (
	"archive/zip"
	"bytes"
	"io"
	"sync"
)

// zipEntryBufferPool 复用 *bytes.Buffer 减少 zip entry 读取时的 GC 压力。
// 各 goroutine 通过 Get/Put 借用；Reset 后归还。注意：调用方在取到 string 后
// 必须确保不再持有 buf 引用（这里 buf.String() 复制了底层字节，安全）。
var zipEntryBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 64<<10))
	},
}

// readZipEntryText 流式读取 zip entry 全部内容并以 string 返回。
// 使用 64KB bufio 缓冲 + Pool 复用 buffer，避免 100MB docx 一次性全量分配。
func readZipEntryText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	br := getBufioReader(rc)
	defer putBufioReader(br)

	buf := zipEntryBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer zipEntryBufferPool.Put(buf)

	if _, err := io.Copy(buf, br); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// readZipEntryBytes 流式读取 zip entry 全部内容并以 []byte 返回。
// 适用于调用方需要原始字节（如 Excel 二进制 sheet）。同样使用 Pool 复用。
func readZipEntryBytes(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	br := getBufioReader(rc)
	defer putBufioReader(br)

	buf := zipEntryBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer zipEntryBufferPool.Put(buf)

	if _, err := io.Copy(buf, br); err != nil {
		return nil, err
	}
	// 必须 copy 一份：buf 即将归还 pool
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}
