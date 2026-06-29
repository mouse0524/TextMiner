package extractor

import (
	"strings"
	"testing"
)

func TestBuildCacheKey(t *testing.T) {
	k1 := buildCacheKey("/tmp/a.txt", 123, 456, false)
	k2 := buildCacheKey("/tmp/a.txt", 123, 456, false)
	if k1 != k2 {
		t.Fatalf("相同输入应产生相同 key: %q vs %q", k1, k2)
	}

	// size 不同 → key 不同（避免 mtime 不变但内容变化的误命中）
	k3 := buildCacheKey("/tmp/a.txt", 123, 457, false)
	if k1 == k3 {
		t.Fatal("size 不同应产生不同 key")
	}

	// ocr 不同 → key 不同
	k4 := buildCacheKey("/tmp/a.txt", 123, 456, true)
	if k1 == k4 {
		t.Fatal("ocr 不同应产生不同 key")
	}

	// 校验 key 格式：4 段以 ':' 分隔
	if strings.Count(k1, ":") != 3 {
		t.Fatalf("key 期望 3 个分隔符，got %q", k1)
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusSuccess != "success" {
		t.Errorf("StatusSuccess=%q", StatusSuccess)
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed=%q", StatusFailed)
	}
	if StatusSkipped != "skipped" {
		t.Errorf("StatusSkipped=%q", StatusSkipped)
	}
}
