package extractor

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTestZip 在临时目录构造一个 zip 文件，files 描述其中的 entry。
func makeTestZip(t *testing.T, name string, files map[string][]byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, name)
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create temp zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, data := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %s: %v", name, err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatalf("write entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return zipPath
}

func TestArchiveExtractor_NormalZip(t *testing.T) {
	zipPath := makeTestZip(t, "normal.zip", map[string][]byte{
		"hello.txt":   []byte("Hello World"),
		"sub/foo.txt": []byte("nested content"),
	})

	e := NewArchiveExtractor("zip")
	files, err := e.extractZip(zipPath, 0)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 file, got 0")
	}
}

func TestArchiveExtractor_ZipSlipBlocked(t *testing.T) {
	// 手工构造一个含 ../escape.txt 的 zip
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "slip.zip")
	f, _ := os.Create(zipPath)
	defer f.Close()

	w := zip.NewWriter(f)
	fw, _ := w.Create("../escape.txt")
	_, _ = fw.Write([]byte("pwned"))
	_ = w.Close()

	e := NewArchiveExtractor("zip")
	// extractZip 通过 SafeReadZipEntry 过滤 ../ 路径
	// 期望该 entry 被跳过（logger.Warnf），不进入返回结果
	files, err := e.extractZip(zipPath, 0)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name, "..") {
			t.Fatalf("zip slip entry 未被过滤: %s", file.Name)
		}
	}
}

func TestArchiveExtractor_FileCountLimit(t *testing.T) {
	// 构造超过 MaxArchiveFileCount 的 zip
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "huge.zip")
	f, _ := os.Create(zipPath)
	defer f.Close()

	w := zip.NewWriter(f)
	for i := 0; i < MaxArchiveFileCount+1; i++ {
		fw, _ := w.Create("file" + string(rune(i%26+'a')) + ".txt")
		_, _ = fw.Write([]byte("x"))
	}
	_ = w.Close()

	e := NewArchiveExtractor("zip")
	_, err := e.extractZip(zipPath, 0)
	if err == nil {
		t.Fatal("超过 MaxArchiveFileCount 应报错")
	}
}

func TestSafeReadLimited_RespectsSingleFileLimit(t *testing.T) {
	// 模拟一个 100 字节的 reader 加上 MaxSingleFileSize+1 的限制
	// MaxSingleFileSize=1<<30，应正常通过
	data := bytes.Repeat([]byte{0x41}, 100)
	got, err := SafeReadLimited(bytes.NewReader(data), nil)
	if err != nil {
		t.Fatalf("SafeReadLimited failed: %v", err)
	}
	if len(got) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(got))
	}
}
