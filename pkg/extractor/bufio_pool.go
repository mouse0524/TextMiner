package extractor

import (
	"bufio"
	"io"
	"sync"
)

// bufioReaderPool 复用 *bufio.Reader（64KB 缓冲）：替换 5 处
// 直接 `bufio.NewReaderSize(f, 64<<10)` 调用，避免每次分配。
//
// 调用方必须遵循 Get-Reset-Put 链：
//
//	br := getBufioReader(f)
//	defer putBufioReader(br)
//	data, _ := io.ReadAll(br)
//
// 借出后只用于当前 goroutine；跨 goroutine 共享 reader 会导致数据错乱。
var bufioReaderPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReaderSize(nil, 64<<10)
	},
}

func getBufioReader(r io.Reader) *bufio.Reader {
	br := bufioReaderPool.Get().(*bufio.Reader)
	br.Reset(r)
	return br
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}
