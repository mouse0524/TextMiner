package extractor

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

// validateFilePath 校验用户提供的文件路径，拒绝 Path Traversal 与可疑输入。
// 接受绝对或相对路径；空字符串或仅空白会被拒绝。
// 注意：仅做静态校验；符号链接跟随由 os.Stat 决定。
func validateFilePath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("文件路径为空")
	}
	cleaned := filepath.Clean(p)
	// 拒绝 NUL 等控制字符
	for _, r := range cleaned {
		if r == 0 {
			return "", fmt.Errorf("文件路径包含 NUL 字符")
		}
	}
	// Windows 上绝对路径（带盘符）合法；仅记录诊断信息
	if runtime.GOOS != "windows" && !filepath.IsAbs(cleaned) && strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("文件路径不允许引用上级目录: %s", p)
	}
	return cleaned, nil
}

const (
	// MaxSingleFileSize 单个压缩包内文件解压后允许的最大字节数（1 GiB）
	MaxSingleFileSize int64 = 1 << 30
	// MaxTotalUncompressed 单次解压允许的最大累计字节数（5 GiB）
	MaxTotalUncompressed int64 = 5 << 30
	// MaxArchiveFileCount 单层压缩包允许的最大文件数
	MaxArchiveFileCount = 10000
	// MaxEmbedDepth Office 文档嵌入文件递归提取的最大深度
	MaxEmbedDepth = 3
)

// SanitizeArchiveName 校验压缩包内文件路径，拒绝 Zip Slip 攻击。
// 返回清理后的相对路径；若包含 `..`、绝对路径或空目录则返回错误。
func SanitizeArchiveName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("压缩包内文件名为空")
	}
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == "." {
		return "", fmt.Errorf("压缩包内文件名为当前目录")
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("压缩包内文件名包含上级目录引用: %s", name)
	}
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("压缩包内文件名为绝对路径: %s", name)
	}
	return clean, nil
}

// CheckZipBomb 校验单个文件大小与累计大小，防止解压炸弹。
// uncompressed 是当前已累计解压字节数（不含本文件），fileSize 是本文件解压后大小。
func CheckZipBomb(uncompressed, fileSize int64) error {
	if fileSize > MaxSingleFileSize {
		return fmt.Errorf("压缩包内单文件超过 %d 字节，跳过 (疑似 zip bomb)", MaxSingleFileSize)
	}
	if uncompressed+fileSize > MaxTotalUncompressed {
		return fmt.Errorf("压缩包累计解压超过 %d 字节，终止 (疑似 zip bomb)", MaxTotalUncompressed)
	}
	return nil
}

// CheckArchiveFileCount 限制单层文件数量
func CheckArchiveFileCount(count int) error {
	if count > MaxArchiveFileCount {
		return fmt.Errorf("压缩包内文件数超过 %d，跳过", MaxArchiveFileCount)
	}
	return nil
}

// SafeReadZipEntry 读取 zip 内部单个文件，自动应用 Zip Slip 校验与 Zip Bomb 防护。
// 返回 (sanitizedName, contentBytes, error)。
func SafeReadZipEntry(file *zip.File, totalSoFar *int64) (string, []byte, error) {
	name, err := SanitizeArchiveName(file.Name)
	if err != nil {
		return "", nil, err
	}
	if file.UncompressedSize64 > uint64(MaxSingleFileSize) {
		return name, nil, fmt.Errorf("跳过过大文件 %s: 声明大小 %d 字节", name, file.UncompressedSize64)
	}
	rc, err := file.Open()
	if err != nil {
		return name, nil, fmt.Errorf("打开压缩包内文件失败 %s: %w", name, err)
	}
	defer rc.Close()

	// 使用 LimitedReader 防止意外超大文件
	limited := &io.LimitedReader{R: rc, N: MaxSingleFileSize + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return name, nil, fmt.Errorf("读取压缩包内文件失败 %s: %w", name, err)
	}
	if int64(len(data)) > MaxSingleFileSize {
		return name, nil, fmt.Errorf("跳过过大文件 %s: 实际大小超过 %d 字节", name, MaxSingleFileSize)
	}
	if totalSoFar != nil {
		*totalSoFar += int64(len(data))
		if *totalSoFar > MaxTotalUncompressed {
			return name, nil, fmt.Errorf("累计解压超过 %d 字节，终止 (疑似 zip bomb)", MaxTotalUncompressed)
		}
	}
	return name, data, nil
}

// SafeReadLimited 通用压缩包 entry 安全读取：使用 io.LimitReader 限制单文件大小，
// 累加 totalSoFar 用于累计限制。适用于 7z/rar/tar/gz/xz/bz2 等格式。
func SafeReadLimited(rc io.Reader, totalSoFar *int64) ([]byte, error) {
	limited := &io.LimitedReader{R: rc, N: MaxSingleFileSize + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("读取压缩包内容失败: %w", err)
	}
	if int64(len(data)) > MaxSingleFileSize {
		return nil, fmt.Errorf("跳过过大文件: 实际大小超过 %d 字节", MaxSingleFileSize)
	}
	if totalSoFar != nil {
		*totalSoFar += int64(len(data))
		if *totalSoFar > MaxTotalUncompressed {
			return nil, fmt.Errorf("累计解压超过 %d 字节，终止 (疑似 zip bomb)", MaxTotalUncompressed)
		}
	}
	return data, nil
}
