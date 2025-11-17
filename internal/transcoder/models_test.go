package transcoder

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

func TestThumbnailTypes(t *testing.T) {
	validTypes := []string{
		models.ThumbnailTypeSingle,
		models.ThumbnailTypeSprite,
		models.ThumbnailTypeAnimated,
	}

	for _, typ := range validTypes {
		if typ == "" {
			t.Errorf("thumbnail type should not be empty")
		}
	}
}

func TestSubtitleFormats(t *testing.T) {
	validFormats := []string{
		models.SubtitleFormatVTT,
		models.SubtitleFormatSRT,
		models.SubtitleFormatASS,
	}

	for _, format := range validFormats {
		if format == "" {
			t.Errorf("subtitle format should not be empty")
		}
	}
}

func TestStreamingTypes(t *testing.T) {
	validTypes := []string{
		models.StreamingTypeProgressive,
		models.StreamingTypeHLS,
		models.StreamingTypeDASH,
	}

	for _, typ := range validTypes {
		if typ == "" {
			t.Errorf("streaming type should not be empty")
		}
	}
}

func TestProfileTypes(t *testing.T) {
	validTypes := []string{
		models.ProfileTypeHLS,
		models.ProfileTypeDASH,
	}

	for _, typ := range validTypes {
		if typ == "" {
			t.Errorf("profile type should not be empty")
		}
	}
}

func TestResolutionProfileValidation(t *testing.T) {
	profiles := []models.ResolutionProfile{
		models.Resolution144p,
		models.Resolution240p,
		models.Resolution360p,
		models.Resolution480p,
		models.Resolution720p,
		models.Resolution1080p,
		models.Resolution4K,
	}

	for _, profile := range profiles {
		// Validate name
		if profile.Name == "" {
			t.Errorf("profile name should not be empty")
		}

		// Validate dimensions
		if profile.Width <= 0 || profile.Height <= 0 {
			t.Errorf("profile %s has invalid dimensions: %dx%d",
				profile.Name, profile.Width, profile.Height)
		}

		// Validate bitrates
		if profile.VideoBitrate <= 0 {
			t.Errorf("profile %s has invalid video bitrate: %d",
				profile.Name, profile.VideoBitrate)
		}

		if profile.AudioBitrate <= 0 {
			t.Errorf("profile %s has invalid audio bitrate: %d",
				profile.Name, profile.AudioBitrate)
		}

		// Validate max/min bitrates if set
		if profile.MaxBitrate > 0 && profile.MaxBitrate <= profile.VideoBitrate {
			t.Errorf("profile %s has invalid max bitrate: %d (should be > %d)",
				profile.Name, profile.MaxBitrate, profile.VideoBitrate)
		}

		if profile.MinBitrate > 0 && profile.MinBitrate >= profile.VideoBitrate {
			t.Errorf("profile %s has invalid min bitrate: %d (should be < %d)",
				profile.Name, profile.MinBitrate, profile.VideoBitrate)
		}

		// Validate aspect ratio (should be ~16:9 for most)
		aspectRatio := float64(profile.Width) / float64(profile.Height)
		expectedAspectRatio := 16.0 / 9.0
		tolerance := 0.1

		if aspectRatio < expectedAspectRatio-tolerance || aspectRatio > expectedAspectRatio+tolerance {
			t.Logf("Warning: profile %s has non-standard aspect ratio: %.2f (expected ~%.2f)",
				profile.Name, aspectRatio, expectedAspectRatio)
		}
	}
}
