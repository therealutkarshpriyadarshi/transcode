package models

// ResolutionProfile defines a resolution and bitrate combination
type ResolutionProfile struct {
	Name         string `json:"name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	VideoBitrate int64  `json:"video_bitrate"`
	AudioBitrate int    `json:"audio_bitrate"`
	MaxBitrate   int64  `json:"max_bitrate,omitempty"`
	MinBitrate   int64  `json:"min_bitrate,omitempty"`
}

// Standard resolution profiles based on industry standards
var (
	// Resolution4K represents 4K/UHD resolution
	Resolution4K = ResolutionProfile{
		Name:         "4K",
		Width:        3840,
		Height:       2160,
		VideoBitrate: 20000000, // 20 Mbps
		AudioBitrate: 192000,   // 192 kbps
		MaxBitrate:   25000000,
		MinBitrate:   15000000,
	}

	// Resolution1080p represents Full HD resolution
	Resolution1080p = ResolutionProfile{
		Name:         "1080p",
		Width:        1920,
		Height:       1080,
		VideoBitrate: 6500000, // 6.5 Mbps
		AudioBitrate: 128000,  // 128 kbps
		MaxBitrate:   8000000,
		MinBitrate:   5000000,
	}

	// Resolution720p represents HD resolution
	Resolution720p = ResolutionProfile{
		Name:         "720p",
		Width:        1280,
		Height:       720,
		VideoBitrate: 3500000, // 3.5 Mbps
		AudioBitrate: 128000,  // 128 kbps
		MaxBitrate:   4000000,
		MinBitrate:   2500000,
	}

	// Resolution480p represents SD resolution
	Resolution480p = ResolutionProfile{
		Name:         "480p",
		Width:        854,
		Height:       480,
		VideoBitrate: 1500000, // 1.5 Mbps
		AudioBitrate: 96000,   // 96 kbps
		MaxBitrate:   2000000,
		MinBitrate:   1200000,
	}

	// Resolution360p represents low-quality mobile resolution
	Resolution360p = ResolutionProfile{
		Name:         "360p",
		Width:        640,
		Height:       360,
		VideoBitrate: 900000, // 900 kbps
		AudioBitrate: 96000,  // 96 kbps
		MaxBitrate:   1200000,
		MinBitrate:   700000,
	}

	// Resolution240p represents very low bandwidth resolution
	Resolution240p = ResolutionProfile{
		Name:         "240p",
		Width:        426,
		Height:       240,
		VideoBitrate: 500000, // 500 kbps
		AudioBitrate: 64000,  // 64 kbps
		MaxBitrate:   700000,
		MinBitrate:   400000,
	}

	// Resolution144p represents minimal bandwidth resolution
	Resolution144p = ResolutionProfile{
		Name:         "144p",
		Width:        256,
		Height:       144,
		VideoBitrate: 200000, // 200 kbps
		AudioBitrate: 64000,  // 64 kbps
		MaxBitrate:   300000,
		MinBitrate:   100000,
	}
)

// ResolutionLadder returns all standard resolution profiles
func ResolutionLadder() []ResolutionProfile {
	return []ResolutionProfile{
		Resolution144p,
		Resolution240p,
		Resolution360p,
		Resolution480p,
		Resolution720p,
		Resolution1080p,
		Resolution4K,
	}
}

// GetResolutionProfile returns a resolution profile by name
func GetResolutionProfile(name string) *ResolutionProfile {
	profiles := map[string]ResolutionProfile{
		"144p":  Resolution144p,
		"240p":  Resolution240p,
		"360p":  Resolution360p,
		"480p":  Resolution480p,
		"720p":  Resolution720p,
		"1080p": Resolution1080p,
		"4K":    Resolution4K,
		"2160p": Resolution4K,
	}

	if profile, ok := profiles[name]; ok {
		return &profile
	}
	return nil
}

// SelectResolutionsForVideo intelligently selects appropriate resolutions
// based on source video dimensions
func SelectResolutionsForVideo(sourceWidth, sourceHeight int) []ResolutionProfile {
	var selected []ResolutionProfile

	ladder := ResolutionLadder()
	for _, profile := range ladder {
		// Only include resolutions <= source resolution
		if profile.Width <= sourceWidth && profile.Height <= sourceHeight {
			selected = append(selected, profile)
		}
	}

	// Always include at least one resolution
	if len(selected) == 0 && len(ladder) > 0 {
		selected = append(selected, ladder[0])
	}

	return selected
}
