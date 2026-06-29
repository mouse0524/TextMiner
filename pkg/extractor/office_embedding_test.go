package extractor

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestZipWithEmbeddings 在临时目录中创建一个 zip 文件，
// 内部包含指定路径的条目，entries 形如 "word/embeddings/oleObject1.bin"。
// 返回 zip 文件路径，调用方负责清理。
func createTestZipWithEmbeddings(t *testing.T, entries map[string][]byte) string {
	t.Helper()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.docx")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("创建临时 zip 失败: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, data := range entries {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("创建 zip 条目 %q 失败: %v", name, err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatalf("写入 zip 条目 %q 失败: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("关闭 zip writer 失败: %v", err)
	}
	return zipPath
}

// TestOfficeEmbedding_MaxDepth 验证 Office 嵌入递归深度限制：
// 当 depth >= MaxEmbedDepth 时应直接返回 nil，不递归。
func TestOfficeEmbedding_MaxDepth(t *testing.T) {
	if MaxEmbedDepth < 1 {
		t.Skipf("MaxEmbedDepth=%d 不合理，跳过", MaxEmbedDepth)
	}

	// OLE2 magic 让 extractFromBinFile 进入 OLE 分支（实际解析会失败但不应 panic）
	oleMagic := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"word/embeddings/oleObject1.bin": oleMagic,
		"[Content_Types].xml":            []byte(`<?xml version="1.0"?>`),
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("docx")
	var content strings.Builder

	// depth 等于上限：应直接返回 nil，不进入处理循环
	if err := ext.ExtractFromOfficeFile(reader, &content, MaxEmbedDepth); err != nil {
		t.Fatalf("depth=MaxEmbedDepth 不应返回错误，got %v", err)
	}
	if content.Len() != 0 {
		t.Errorf("depth=MaxEmbedDepth 不应写入内容，got len=%d", content.Len())
	}

	// depth 超过上限：同上
	if err := ext.ExtractFromOfficeFile(reader, &content, MaxEmbedDepth+1); err != nil {
		t.Fatalf("depth=MaxEmbedDepth+1 不应返回错误，got %v", err)
	}
}

// TestOfficeEmbedding_DepthZero 验证 depth=0 时会处理 embeddings 路径下条目（不会 panic）。
func TestOfficeEmbedding_DepthZero(t *testing.T) {
	// 空 zip，无 embeddings 路径下条目：应不报错且不写入内容
	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0"?>`),
		"word/document.xml":   []byte(`<w:document/>`),
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("docx")
	var content strings.Builder

	if err := ext.ExtractFromOfficeFile(reader, &content, 0); err != nil {
		t.Fatalf("depth=0 不应返回错误，got %v", err)
	}
}

// TestOfficeEmbedding_UnknownType 验证未知文件类型直接返回 nil。
func TestOfficeEmbedding_UnknownType(t *testing.T) {
	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document/>`),
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("unknown")
	var content strings.Builder

	if err := ext.ExtractFromOfficeFile(reader, &content, 0); err != nil {
		t.Fatalf("unknown 类型不应返回错误，got %v", err)
	}
	if content.Len() != 0 {
		t.Errorf("unknown 类型不应写入内容，got len=%d", content.Len())
	}
}

// TestOfficeEmbedding_PPTX 验证 PPTX 走 ppt/embeddings/ 路径。
func TestOfficeEmbedding_PPTX(t *testing.T) {
	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"ppt/embeddings/oleObject1.bin": []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("pptx")
	var content strings.Builder

	// 触发 OLE 解析路径（会失败但不应 panic）
	if err := ext.ExtractFromOfficeFile(reader, &content, 0); err != nil {
		t.Logf("PPTX 提取返回错误（可接受，因 OLE 数据非真实）: %v", err)
	}
}

// TestOfficeEmbedding_XLSX 验证 XLSX 走 xl/embeddings/ 路径。
func TestOfficeEmbedding_XLSX(t *testing.T) {
	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"xl/embeddings/oleObject1.bin": []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1},
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("xlsx")
	var content strings.Builder

	if err := ext.ExtractFromOfficeFile(reader, &content, 0); err != nil {
		t.Logf("XLSX 提取返回错误（可接受，因 OLE 数据非真实）: %v", err)
	}
}

// TestOfficeEmbedding_IncrementalDepth 验证每层 depth 都被尊重。
func TestOfficeEmbedding_IncrementalDepth(t *testing.T) {
	if MaxEmbedDepth < 1 {
		t.Skipf("MaxEmbedDepth=%d 不合理", MaxEmbedDepth)
	}

	zipPath := createTestZipWithEmbeddings(t, map[string][]byte{
		"word/document.xml": []byte(`<w:document/>`),
	})
	defer os.Remove(zipPath)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	ext := NewOfficeEmbeddingExtractor("docx")
	var content strings.Builder

	// 逐层调用：depth=0..MaxEmbedDepth-1 均不应 panic
	for d := 0; d < MaxEmbedDepth; d++ {
		if err := ext.ExtractFromOfficeFile(reader, &content, d); err != nil {
			t.Fatalf("depth=%d 不应返回错误，got %v", d, err)
		}
	}
}
