package extractor

import (
	"sync"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
)

func TestLRUCache_Eviction(t *testing.T) {
	cache, err := lru.New[string, string](3)
	if err != nil {
		t.Fatalf("lru.New failed: %v", err)
	}

	cache.Add("k1", "v1")
	cache.Add("k2", "v2")
	cache.Add("k3", "v3")

	if cache.Len() != 3 {
		t.Fatalf("expected len=3, got %d", cache.Len())
	}

	// 触发 k1 变为最近最少使用
	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected k1=v1, got (%q, %v)", v, ok)
	}
	// 触发 k2 变为最近最少使用
	if v, ok := cache.Get("k2"); !ok || v != "v2" {
		t.Fatalf("expected k2=v2, got (%q, %v)", v, ok)
	}

	// 添加 k4 应驱逐 k3
	cache.Add("k4", "v4")

	if _, ok := cache.Get("k3"); ok {
		t.Fatal("k3 应被驱逐")
	}
	if _, ok := cache.Get("k1"); !ok {
		t.Fatal("k1 不应被驱逐（最近使用）")
	}
	if _, ok := cache.Get("k4"); !ok {
		t.Fatal("k4 应保留")
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache, _ := lru.New[string, string](10)
	cache.Add("a", "1")
	cache.Add("b", "2")

	if cache.Len() != 2 {
		t.Fatalf("expected 2, got %d", cache.Len())
	}
	clearCache(cache)
	if cache.Len() != 0 {
		t.Fatalf("after clear, expected 0, got %d", cache.Len())
	}
}

func TestLRUCache_GetSize(t *testing.T) {
	cache, _ := lru.New[string, string](10)
	if got := getCacheSize(cache); got != 0 {
		t.Fatalf("empty cache size should be 0, got %d", got)
	}
	cache.Add("x", "1")
	cache.Add("y", "2")
	if got := getCacheSize(cache); got != 2 {
		t.Fatalf("expected size 2, got %d", got)
	}
}

func TestLRUCache_ConcurrentSafe(t *testing.T) {
	cache, _ := lru.New[string, string](100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := "key"
				cache.Add(key, "v")
				_, _ = cache.Get(key)
			}
		}(i)
	}
	wg.Wait()
}
