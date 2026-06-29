package extractor

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeArchiveName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"normal", "docs/readme.txt", "docs/readme.txt", false},
		{"nested", "a/b/c/file.go", "a/b/c/file.go", false},
		{"current", "./file.txt", "file.txt", false},
		{"zip-slip-parent", "../etc/passwd", "", true},
		{"zip-slip-deep-traverses-out", "a/b/c/d/../../../../../escape", "", true},
		{"absolute-unix", "/etc/passwd", "", true},
		{"absolute-windows", `C:\Windows\System32`, "", true},
		{"empty", "", "", true},
		{"dot", ".", "", true},
		{"zip-slip-prefix", "../malicious.txt", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SanitizeArchiveName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望错误，但 got=%q err=nil", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestCheckZipBomb(t *testing.T) {
	tests := []struct {
		name        string
		accumulated int64
		fileSize    int64
		wantErr     bool
	}{
		{"normal", 0, 1024, false},
		{"single-over-1gib", 0, MaxSingleFileSize + 1, true},
		{"cumulative-over-5gib", MaxTotalUncompressed - 100, 200, true},
		{"cumulative-under-limit", 1024 * 1024, 1024 * 1024, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckZipBomb(tc.accumulated, tc.fileSize)
			if tc.wantErr && err == nil {
				t.Fatal("期望错误但 got=nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("意外错误: %v", err)
			}
		})
	}
}

func TestCheckArchiveFileCount(t *testing.T) {
	if err := CheckArchiveFileCount(MaxArchiveFileCount); err != nil {
		t.Fatalf("恰等上限不应报错: %v", err)
	}
	if err := CheckArchiveFileCount(MaxArchiveFileCount + 1); err == nil {
		t.Fatal("超过上限应报错")
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid-relative", "test.txt", false},
		{"valid-absolute", filepath.Join("root", "test.txt"), false},
		{"empty", "", true},
		{"null-byte", "test\x00.txt", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateFilePath(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望错误，got=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if got == "" {
				t.Fatal("结果不应为空")
			}
		})
	}
}

func TestMapExtensionToMimeType(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"pdf", "application/pdf"},
		{"mp3", "audio/mpeg"},
		{"mp4", "video/mp4"},
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"tar.gz", "application/x-tar-gz"},
		{"tar.bz2", "application/x-tar-bz2"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tc := range tests {
		got := MapExtensionToMimeType(tc.ext)
		if got != tc.want {
			t.Errorf("MapExtensionToMimeType(%q)=%q, want %q", tc.ext, got, tc.want)
		}
	}
}

func TestResolveMimeType(t *testing.T) {
	// 关键场景：ext[1:] 会 panic 的无扩展名文件
	got := resolveMimeType("noext")
	if got != "application/octet-stream" {
		t.Errorf("无扩展名应返回 octet-stream，got %q", got)
	}

	got = resolveMimeType("test.pdf")
	if got != "application/pdf" {
		t.Errorf("test.pdf 应返回 application/pdf，got %q", got)
	}

	got = resolveMimeType("TEST.PDF")
	if !strings.EqualFold(got, "application/pdf") {
		t.Errorf("大写扩展名应被小写化，got %q", got)
	}
}
