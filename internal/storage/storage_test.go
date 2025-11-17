package storage

import (
	"testing"
)

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filePath    string
		wantType    string
	}{
		{"video.mp4", "video/mp4"},
		{"video.mov", "video/quicktime"},
		{"video.avi", "video/x-msvideo"},
		{"video.mkv", "video/x-matroska"},
		{"video.webm", "video/webm"},
		{"playlist.m3u8", "application/vnd.apple.mpegurl"},
		{"segment.ts", "video/mp2t"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			contentType := getContentType(tt.filePath)
			if contentType != tt.wantType {
				t.Errorf("getContentType(%q) = %q, want %q", tt.filePath, contentType, tt.wantType)
			}
		})
	}
}
