package extractor

import (
	"fmt"
)

// wrapError 模拟 fmt.Errorf("context: %w", ErrEncrypted)。
// 单独写在测试文件中以避免对核心代码的影响。
func wrapError(base error, ctx string) error {
	return fmt.Errorf("%s: %w", ctx, base)
}
