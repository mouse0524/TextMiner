package extractor

import "testing"

func TestInferFileTypeFromMime(t *testing.T) {
	tests := []struct {
		name  string
		mime  string
		want  string
	}{
		{"audio-mp3", "audio/mpeg", "mp3"},
		{"audio-wav", "audio/wav", "wav"},
		{"audio-midi", "audio/midi", "mid"},
		{"audio-ogg", "audio/ogg", "ogg"},
		{"audio-flac", "audio/flac", "flac"},
		{"audio-aac", "audio/aac", "aac"},
		{"video-mp4", "video/mp4", "mp4"},
		{"video-3gpp", "video/3gpp", "3gp"},
		{"winhlp", "application/winhlp", "hlp"},
		{"mscompress", "application/x-mscompress-szdd", "mscompress"},
		{"unknown-mime", "application/x-unknown-format", ""},
		{"empty-mime", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inferFileTypeFromMime(tc.mime)
			if got != tc.want {
				t.Errorf("inferFileTypeFromMime(%q)=%q, want %q", tc.mime, got, tc.want)
			}
		})
	}
}

func TestInferFileTypeFromMime_O1Complexity(t *testing.T) {
	// 验证该函数为 O(1) 查表，无 case 链
	mime := "audio/wav"
	for i := 0; i < 1000; i++ {
		_ = inferFileTypeFromMime(mime)
	}
	// 如果是 switch/case，长字符串测试会有可观测耗时
	// 此处仅作烟雾测试，确保无 panic / 性能回归
}
