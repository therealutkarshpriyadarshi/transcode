package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

// Phase 2 Repository Methods for Thumbnails, Subtitles, Streaming Profiles, and Audio Tracks

// Thumbnails

// CreateThumbnail creates a new thumbnail record
func (r *Repository) CreateThumbnail(ctx context.Context, thumbnail *models.Thumbnail) error {
	if thumbnail.ID == "" {
		thumbnail.ID = uuid.New().String()
	}

	query := `
		INSERT INTO thumbnails (id, video_id, thumbnail_type, url, path, width, height,
		                        timestamp, sprite_columns, sprite_rows, interval_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		thumbnail.ID, thumbnail.VideoID, thumbnail.ThumbnailType, thumbnail.URL, thumbnail.Path,
		thumbnail.Width, thumbnail.Height, thumbnail.Timestamp, thumbnail.SpriteColumns,
		thumbnail.SpriteRows, thumbnail.IntervalSeconds,
	).Scan(&thumbnail.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create thumbnail: %w", err)
	}

	return nil
}

// GetThumbnailsByVideoID retrieves all thumbnails for a video
func (r *Repository) GetThumbnailsByVideoID(ctx context.Context, videoID string) ([]*models.Thumbnail, error) {
	query := `
		SELECT id, video_id, thumbnail_type, url, path, width, height,
		       timestamp, sprite_columns, sprite_rows, interval_seconds, created_at
		FROM thumbnails
		WHERE video_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query thumbnails: %w", err)
	}
	defer rows.Close()

	var thumbnails []*models.Thumbnail
	for rows.Next() {
		var thumbnail models.Thumbnail
		err := rows.Scan(
			&thumbnail.ID, &thumbnail.VideoID, &thumbnail.ThumbnailType, &thumbnail.URL,
			&thumbnail.Path, &thumbnail.Width, &thumbnail.Height, &thumbnail.Timestamp,
			&thumbnail.SpriteColumns, &thumbnail.SpriteRows, &thumbnail.IntervalSeconds,
			&thumbnail.CreatedAt,
		)
		if err != nil {
			continue
		}
		thumbnails = append(thumbnails, &thumbnail)
	}

	return thumbnails, nil
}

// Subtitles

// CreateSubtitle creates a new subtitle record
func (r *Repository) CreateSubtitle(ctx context.Context, subtitle *models.Subtitle) error {
	if subtitle.ID == "" {
		subtitle.ID = uuid.New().String()
	}

	query := `
		INSERT INTO subtitles (id, video_id, language, label, format, url, path, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		subtitle.ID, subtitle.VideoID, subtitle.Language, subtitle.Label,
		subtitle.Format, subtitle.URL, subtitle.Path, subtitle.IsDefault,
	).Scan(&subtitle.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create subtitle: %w", err)
	}

	return nil
}

// GetSubtitlesByVideoID retrieves all subtitles for a video
func (r *Repository) GetSubtitlesByVideoID(ctx context.Context, videoID string) ([]*models.Subtitle, error) {
	query := `
		SELECT id, video_id, language, label, format, url, path, is_default, created_at
		FROM subtitles
		WHERE video_id = $1
		ORDER BY is_default DESC, language ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitles: %w", err)
	}
	defer rows.Close()

	var subtitles []*models.Subtitle
	for rows.Next() {
		var subtitle models.Subtitle
		err := rows.Scan(
			&subtitle.ID, &subtitle.VideoID, &subtitle.Language, &subtitle.Label,
			&subtitle.Format, &subtitle.URL, &subtitle.Path, &subtitle.IsDefault,
			&subtitle.CreatedAt,
		)
		if err != nil {
			continue
		}
		subtitles = append(subtitles, &subtitle)
	}

	return subtitles, nil
}

// Streaming Profiles

// CreateStreamingProfile creates a new streaming profile record
func (r *Repository) CreateStreamingProfile(ctx context.Context, profile *models.StreamingProfile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	query := `
		INSERT INTO streaming_profiles (id, video_id, job_id, profile_type, master_manifest_url,
		                                 master_manifest_path, variant_count, audio_only)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		profile.ID, profile.VideoID, profile.JobID, profile.ProfileType,
		profile.MasterManifestURL, profile.MasterManifestPath, profile.VariantCount,
		profile.AudioOnly,
	).Scan(&profile.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create streaming profile: %w", err)
	}

	return nil
}

// GetStreamingProfilesByVideoID retrieves all streaming profiles for a video
func (r *Repository) GetStreamingProfilesByVideoID(ctx context.Context, videoID string) ([]*models.StreamingProfile, error) {
	query := `
		SELECT id, video_id, job_id, profile_type, master_manifest_url,
		       master_manifest_path, variant_count, audio_only, created_at
		FROM streaming_profiles
		WHERE video_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query streaming profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*models.StreamingProfile
	for rows.Next() {
		var profile models.StreamingProfile
		err := rows.Scan(
			&profile.ID, &profile.VideoID, &profile.JobID, &profile.ProfileType,
			&profile.MasterManifestURL, &profile.MasterManifestPath, &profile.VariantCount,
			&profile.AudioOnly, &profile.CreatedAt,
		)
		if err != nil {
			continue
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

// GetStreamingProfile retrieves a streaming profile by ID
func (r *Repository) GetStreamingProfile(ctx context.Context, id string) (*models.StreamingProfile, error) {
	var profile models.StreamingProfile

	query := `
		SELECT id, video_id, job_id, profile_type, master_manifest_url,
		       master_manifest_path, variant_count, audio_only, created_at
		FROM streaming_profiles
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.VideoID, &profile.JobID, &profile.ProfileType,
		&profile.MasterManifestURL, &profile.MasterManifestPath, &profile.VariantCount,
		&profile.AudioOnly, &profile.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("streaming profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get streaming profile: %w", err)
	}

	return &profile, nil
}

// Audio Tracks

// CreateAudioTrack creates a new audio track record
func (r *Repository) CreateAudioTrack(ctx context.Context, track *models.AudioTrack) error {
	if track.ID == "" {
		track.ID = uuid.New().String()
	}

	query := `
		INSERT INTO audio_tracks (id, video_id, streaming_profile_id, language, label,
		                          codec, bitrate, channels, sample_rate, url, path, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		track.ID, track.VideoID, track.StreamingProfileID, track.Language, track.Label,
		track.Codec, track.Bitrate, track.Channels, track.SampleRate, track.URL,
		track.Path, track.IsDefault,
	).Scan(&track.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create audio track: %w", err)
	}

	return nil
}

// GetAudioTracksByVideoID retrieves all audio tracks for a video
func (r *Repository) GetAudioTracksByVideoID(ctx context.Context, videoID string) ([]*models.AudioTrack, error) {
	query := `
		SELECT id, video_id, streaming_profile_id, language, label, codec, bitrate,
		       channels, sample_rate, url, path, is_default, created_at
		FROM audio_tracks
		WHERE video_id = $1
		ORDER BY is_default DESC, language ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audio tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*models.AudioTrack
	for rows.Next() {
		var track models.AudioTrack
		err := rows.Scan(
			&track.ID, &track.VideoID, &track.StreamingProfileID, &track.Language,
			&track.Label, &track.Codec, &track.Bitrate, &track.Channels, &track.SampleRate,
			&track.URL, &track.Path, &track.IsDefault, &track.CreatedAt,
		)
		if err != nil {
			continue
		}
		tracks = append(tracks, &track)
	}

	return tracks, nil
}

// GetAudioTracksByProfileID retrieves all audio tracks for a streaming profile
func (r *Repository) GetAudioTracksByProfileID(ctx context.Context, profileID string) ([]*models.AudioTrack, error) {
	query := `
		SELECT id, video_id, streaming_profile_id, language, label, codec, bitrate,
		       channels, sample_rate, url, path, is_default, created_at
		FROM audio_tracks
		WHERE streaming_profile_id = $1
		ORDER BY is_default DESC, language ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audio tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*models.AudioTrack
	for rows.Next() {
		var track models.AudioTrack
		err := rows.Scan(
			&track.ID, &track.VideoID, &track.StreamingProfileID, &track.Language,
			&track.Label, &track.Codec, &track.Bitrate, &track.Channels, &track.SampleRate,
			&track.URL, &track.Path, &track.IsDefault, &track.CreatedAt,
		)
		if err != nil {
			continue
		}
		tracks = append(tracks, &track)
	}

	return tracks, nil
}
