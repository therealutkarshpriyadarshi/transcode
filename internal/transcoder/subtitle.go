package transcoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// SubtitleInfo holds subtitle track information
type SubtitleInfo struct {
	Index    int
	Codec    string
	Language string
	Title    string
	Format   string
}

// SubtitleExtractOptions holds options for subtitle extraction
type SubtitleExtractOptions struct {
	InputPath  string
	OutputDir  string
	Format     string   // Output format: "vtt", "srt", "ass" (default: vtt)
	TrackIndex int      // Specific track to extract (-1 for all)
	Languages  []string // Filter by languages (empty for all)
}

// BurnSubtitleOptions holds options for burning subtitles into video
type BurnSubtitleOptions struct {
	InputPath    string
	SubtitlePath string
	OutputPath   string
	SubtitleIndex int    // Index of subtitle stream in input video
	FontName     string
	FontSize     int
	PrimaryColor string
}

// SubtitleExtractionResult holds the result of subtitle extraction
type SubtitleExtractionResult struct {
	Subtitles []ExtractedSubtitle
}

// ExtractedSubtitle represents a single extracted subtitle file
type ExtractedSubtitle struct {
	TrackIndex int
	Language   string
	Format     string
	OutputPath string
}

// ExtractSubtitleInfo extracts subtitle stream information from a video
func (f *FFmpeg) ExtractSubtitleInfo(ctx context.Context, inputPath string) ([]SubtitleInfo, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffprobePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	var metadata struct {
		Streams []struct {
			Index       int               `json:"index"`
			CodecName   string            `json:"codec_name"`
			CodecType   string            `json:"codec_type"`
			Tags        map[string]string `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var subtitles []SubtitleInfo
	for i, stream := range metadata.Streams {
		if stream.CodecType == "subtitle" {
			info := SubtitleInfo{
				Index:    i,
				Codec:    stream.CodecName,
				Language: stream.Tags["language"],
				Title:    stream.Tags["title"],
				Format:   stream.CodecName,
			}
			subtitles = append(subtitles, info)
		}
	}

	return subtitles, nil
}

// ExtractSubtitles extracts subtitle tracks from a video
func (f *FFmpeg) ExtractSubtitles(ctx context.Context, opts SubtitleExtractOptions) (*SubtitleExtractionResult, error) {
	// Set defaults
	if opts.Format == "" {
		opts.Format = "vtt"
	}

	// Get subtitle info
	subtitleInfo, err := f.ExtractSubtitleInfo(ctx, opts.InputPath)
	if err != nil {
		return nil, err
	}

	if len(subtitleInfo) == 0 {
		return &SubtitleExtractionResult{Subtitles: []ExtractedSubtitle{}}, nil
	}

	result := &SubtitleExtractionResult{
		Subtitles: make([]ExtractedSubtitle, 0),
	}

	// Extract each subtitle track
	for _, info := range subtitleInfo {
		// Filter by track index if specified
		if opts.TrackIndex >= 0 && info.Index != opts.TrackIndex {
			continue
		}

		// Filter by language if specified
		if len(opts.Languages) > 0 {
			found := false
			for _, lang := range opts.Languages {
				if strings.EqualFold(info.Language, lang) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Generate output filename
		language := info.Language
		if language == "" {
			language = "unknown"
		}
		outputFilename := fmt.Sprintf("subtitle_%s_%d.%s", language, info.Index, opts.Format)
		outputPath := filepath.Join(opts.OutputDir, outputFilename)

		// Extract subtitle
		args := []string{
			"-i", opts.InputPath,
			"-map", fmt.Sprintf("0:s:%d", info.Index),
		}

		// Convert format if needed
		if opts.Format != info.Format {
			args = append(args, "-c:s", opts.Format)
		} else {
			args = append(args, "-c:s", "copy")
		}

		args = append(args, "-y", outputPath)

		cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Some subtitle formats might not be extractable, continue with next
			continue
		}

		result.Subtitles = append(result.Subtitles, ExtractedSubtitle{
			TrackIndex: info.Index,
			Language:   info.Language,
			Format:     opts.Format,
			OutputPath: outputPath,
		})
	}

	return result, nil
}

// BurnSubtitles burns subtitles into the video (hardcoded subtitles)
func (f *FFmpeg) BurnSubtitles(ctx context.Context, opts BurnSubtitleOptions) error {
	if opts.FontSize <= 0 {
		opts.FontSize = 24
	}

	var subtitleFilter string

	if opts.SubtitlePath != "" {
		// External subtitle file
		// Escape path for FFmpeg filter
		escapedPath := strings.ReplaceAll(opts.SubtitlePath, "\\", "\\\\")
		escapedPath = strings.ReplaceAll(escapedPath, ":", "\\:")
		subtitleFilter = fmt.Sprintf("subtitles=%s", escapedPath)
	} else {
		// Embedded subtitle track
		subtitleFilter = fmt.Sprintf("subtitles=%s:si=%d", opts.InputPath, opts.SubtitleIndex)
	}

	// Add font customization if specified
	if opts.FontName != "" {
		subtitleFilter += fmt.Sprintf(":force_style='FontName=%s,FontSize=%d'", opts.FontName, opts.FontSize)
	}

	if opts.PrimaryColor != "" {
		subtitleFilter += fmt.Sprintf(":force_style='PrimaryColour=%s'", opts.PrimaryColor)
	}

	args := []string{
		"-i", opts.InputPath,
		"-vf", subtitleFilter,
		"-c:a", "copy", // Copy audio without re-encoding
		"-y",
		opts.OutputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("subtitle burning failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// ConvertSubtitleFormat converts a subtitle file from one format to another
func (f *FFmpeg) ConvertSubtitleFormat(ctx context.Context, inputPath, outputPath, outputFormat string) error {
	args := []string{
		"-i", inputPath,
		"-c:s", outputFormat,
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("subtitle conversion failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
