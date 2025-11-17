package transcoder

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestGetResolutionProfile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *models.ResolutionProfile
	}{
		{
			name:     "720p resolution",
			input:    "720p",
			expected: &models.Resolution720p,
		},
		{
			name:     "1080p resolution",
			input:    "1080p",
			expected: &models.Resolution1080p,
		},
		{
			name:     "4K resolution",
			input:    "4K",
			expected: &models.Resolution4K,
		},
		{
			name:     "2160p resolution (4K alias)",
			input:    "2160p",
			expected: &models.Resolution4K,
		},
		{
			name:     "invalid resolution",
			input:    "invalid",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.GetResolutionProfile(tt.input)
			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
			}
			if tt.expected != nil && result != nil {
				if result.Name != tt.expected.Name {
					t.Errorf("expected name %s, got %s", tt.expected.Name, result.Name)
				}
			}
		})
	}
}

func TestSelectResolutionsForVideo(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		height       int
		minExpected  int
		maxExpected  int
		shouldInclude string
	}{
		{
			name:         "4K source video",
			width:        3840,
			height:       2160,
			minExpected:  5,
			maxExpected:  7,
			shouldInclude: "1080p",
		},
		{
			name:         "1080p source video",
			width:        1920,
			height:       1080,
			minExpected:  4,
			maxExpected:  6,
			shouldInclude: "720p",
		},
		{
			name:         "720p source video",
			width:        1280,
			height:       720,
			minExpected:  3,
			maxExpected:  5,
			shouldInclude: "480p",
		},
		{
			name:         "480p source video",
			width:        854,
			height:       480,
			minExpected:  2,
			maxExpected:  4,
			shouldInclude: "360p",
		},
		{
			name:         "very low resolution",
			width:        320,
			height:       180,
			minExpected:  1,
			maxExpected:  2,
			shouldInclude: "144p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.SelectResolutionsForVideo(tt.width, tt.height)

			if len(result) < tt.minExpected || len(result) > tt.maxExpected {
				t.Errorf("expected %d-%d resolutions, got %d", tt.minExpected, tt.maxExpected, len(result))
			}

			found := false
			for _, res := range result {
				if res.Name == tt.shouldInclude {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected to include %s resolution", tt.shouldInclude)
			}

			// Verify all resolutions are <= source
			for _, res := range result {
				if res.Width > tt.width || res.Height > tt.height {
					t.Errorf("resolution %s (%dx%d) exceeds source (%dx%d)",
						res.Name, res.Width, res.Height, tt.width, tt.height)
				}
			}
		})
	}
}

func TestResolutionLadder(t *testing.T) {
	ladder := models.ResolutionLadder()

	if len(ladder) != 7 {
		t.Errorf("expected 7 standard resolutions, got %d", len(ladder))
	}

	// Verify ladder is sorted from lowest to highest
	for i := 1; i < len(ladder); i++ {
		if ladder[i].Width < ladder[i-1].Width {
			t.Errorf("ladder not sorted: %s (%d) comes after %s (%d)",
				ladder[i].Name, ladder[i].Width, ladder[i-1].Name, ladder[i-1].Width)
		}
	}

	// Verify all resolutions have valid bitrates
	for _, res := range ladder {
		if res.VideoBitrate <= 0 {
			t.Errorf("resolution %s has invalid video bitrate: %d", res.Name, res.VideoBitrate)
		}
		if res.AudioBitrate <= 0 {
			t.Errorf("resolution %s has invalid audio bitrate: %d", res.Name, res.AudioBitrate)
		}
	}
}
