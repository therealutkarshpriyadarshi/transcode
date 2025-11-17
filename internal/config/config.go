package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Storage  StorageConfig
	Queue    QueueConfig
	Transcoder TranscoderConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// StorageConfig holds object storage configuration
type StorageConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	UseSSL          bool
}

// QueueConfig holds message queue configuration
type QueueConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Vhost    string
}

// TranscoderConfig holds transcoding configuration
type TranscoderConfig struct {
	WorkerCount        int
	TempDir            string
	FFmpegPath         string
	FFprobePath        string
	MaxConcurrent      int
	ChunkSize          int64
	// Phase 4: GPU Acceleration
	EnableGPU          bool
	GPUDeviceIndex     int
	EnableTwoPass      bool
	// Phase 4: Performance Optimization
	EnableCache        bool
	CacheTTL           time.Duration
	ParallelUpload     bool
	UploadPartSize     int64
	MaxConcurrentParts int
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.readTimeout", "30s")
	viper.SetDefault("server.writeTimeout", "30s")
	viper.SetDefault("server.shutdownTimeout", "10s")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "transcode")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.maxConns", 25)
	viper.SetDefault("database.minConns", 5)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Storage defaults
	viper.SetDefault("storage.endpoint", "localhost:9000")
	viper.SetDefault("storage.accessKeyID", "minioadmin")
	viper.SetDefault("storage.secretAccessKey", "minioadmin")
	viper.SetDefault("storage.bucketName", "videos")
	viper.SetDefault("storage.region", "us-east-1")
	viper.SetDefault("storage.useSSL", false)

	// Queue defaults
	viper.SetDefault("queue.host", "localhost")
	viper.SetDefault("queue.port", 5672)
	viper.SetDefault("queue.user", "guest")
	viper.SetDefault("queue.password", "guest")
	viper.SetDefault("queue.vhost", "/")

	// Transcoder defaults
	viper.SetDefault("transcoder.workerCount", 2)
	viper.SetDefault("transcoder.tempDir", "/tmp/transcode")
	viper.SetDefault("transcoder.ffmpegPath", "ffmpeg")
	viper.SetDefault("transcoder.ffprobePath", "ffprobe")
	viper.SetDefault("transcoder.maxConcurrent", 4)
	viper.SetDefault("transcoder.chunkSize", 5*1024*1024) // 5MB
	// Phase 4: GPU Acceleration defaults
	viper.SetDefault("transcoder.enableGPU", true)
	viper.SetDefault("transcoder.gpuDeviceIndex", -1)
	viper.SetDefault("transcoder.enableTwoPass", false)
	// Phase 4: Performance Optimization defaults
	viper.SetDefault("transcoder.enableCache", true)
	viper.SetDefault("transcoder.cacheTTL", "5m")
	viper.SetDefault("transcoder.parallelUpload", true)
	viper.SetDefault("transcoder.uploadPartSize", 10*1024*1024) // 10MB
	viper.SetDefault("transcoder.maxConcurrentParts", 10)
}
