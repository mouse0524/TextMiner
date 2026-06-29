package extractor

import "testing"

// TestCloseOcrProcessor_Idempotent 验证 CloseOcrProcessor 可重复调用不 panic。
// 实际场景中，如果 Close 后再次 Close，CR 引擎内部 Destroy 已被 Go OCR 库
// 自身视为幂等（Destroy 文档为 close-once），但本项目用 nil 化保证绝对幂等。
func TestCloseOcrProcessor_Idempotent(t *testing.T) {
	// 第一次 Close：实例可能为 nil（未初始化），应不 panic
	if err := CloseOcrProcessor(); err != nil {
		t.Logf("首次 Close 返回 err=%v (可接受，未初始化的实例)", err)
	}

	// 第二次 Close：必须不 panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("重复 Close 触发 panic: %v", r)
		}
	}()
	if err := CloseOcrProcessor(); err != nil {
		t.Logf("二次 Close 返回 err=%v", err)
	}
}

func TestErrEncryptedSentinel(t *testing.T) {
	if ErrEncrypted == nil {
		t.Fatal("ErrEncrypted 不应为 nil")
	}
	if ErrEncrypted.Error() == "" {
		t.Fatal("ErrEncrypted.Error() 不应为空")
	}
	// errors.Is 应当能识别包装过的 ErrEncrypted
	wrapped := wrapError(ErrEncrypted, "test context")
	if !isEncryptedError(wrapped) {
		t.Fatal("errors.Is 应能识别包装过的 ErrEncrypted")
	}
}

func isEncryptedError(err error) bool {
	if err == nil {
		return false
	}
	for err != nil {
		if err == ErrEncrypted {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return false
}
