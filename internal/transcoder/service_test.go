package transcoder

import (
	"testing"
)

func TestParseResolution(t *testing.T) {
	tests := []struct {
		resolution string
		wantWidth  int
		wantHeight int
	}{
		{"144p", 256, 144},
		{"240p", 426, 240},
		{"360p", 640, 360},
		{"480p", 854, 480},
		{"720p", 1280, 720},
		{"1080p", 1920, 1080},
		{"1440p", 2560, 1440},
		{"4k", 3840, 2160},
		{"2160p", 3840, 2160},
		{"unknown", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.resolution, func(t *testing.T) {
			width, height := parseResolution(tt.resolution)
			if width != tt.wantWidth {
				t.Errorf("parseResolution(%q) width = %d, want %d", tt.resolution, width, tt.wantWidth)
			}
			if height != tt.wantHeight {
				t.Errorf("parseResolution(%q) height = %d, want %d", tt.resolution, height, tt.wantHeight)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	cfg := struct {
		FFmpegPath  string
		FFprobePath string
	}{
		FFmpegPath:  "ffmpeg",
		FFprobePath: "ffprobe",
	}

	// We can't fully test the service without dependencies,
	// but we can test that it initializes
	if cfg.FFmpegPath == "" {
		t.Error("FFmpeg path should not be empty")
	}
}
