package transcoder

import (
	"context"
	"testing"
	"time"
)

func TestNewGPUManager(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	if gm == nil {
		t.Fatal("GPUManager should not be nil")
	}

	if gm.ffmpegPath != "ffmpeg" {
		t.Errorf("Expected ffmpegPath to be 'ffmpeg', got %s", gm.ffmpegPath)
	}

	if gm.capability == nil {
		t.Fatal("capability should not be nil")
	}
}

func TestGPUManager_GetCapability(t *testing.T) {
	gm := NewGPUManager("ffmpeg")
	cap := gm.GetCapability()

	if cap == nil {
		t.Fatal("GetCapability should not return nil")
	}

	// Check that last checked time is recent
	if time.Since(cap.LastChecked) > time.Minute {
		t.Error("LastChecked should be recent")
	}
}

func TestGPUManager_IsAvailable(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	// This test will pass or fail depending on whether GPU is available
	available := gm.IsAvailable()

	t.Logf("GPU available: %v", available)

	cap := gm.GetCapability()
	if available != cap.Available || available != cap.NVENCSupported {
		t.Error("IsAvailable should match capability status")
	}
}

func TestGPUManager_GetOptimalCodec(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	tests := []struct {
		name           string
		requestedCodec string
		useGPU         bool
		expectedCPU    string
		expectedGPU    string
	}{
		{
			name:           "H264 CPU",
			requestedCodec: "h264",
			useGPU:         false,
			expectedCPU:    "libx264",
		},
		{
			name:           "H264 GPU",
			requestedCodec: "h264",
			useGPU:         true,
			expectedGPU:    "h264_nvenc",
		},
		{
			name:           "H265 CPU",
			requestedCodec: "h265",
			useGPU:         false,
			expectedCPU:    "libx265",
		},
		{
			name:           "H265 GPU",
			requestedCodec: "h265",
			useGPU:         true,
			expectedGPU:    "hevc_nvenc",
		},
		{
			name:           "VP9 CPU",
			requestedCodec: "vp9",
			useGPU:         false,
			expectedCPU:    "libvpx-vp9",
		},
		{
			name:           "Default CPU",
			requestedCodec: "unknown",
			useGPU:         false,
			expectedCPU:    "libx264",
		},
		{
			name:           "Default GPU",
			requestedCodec: "unknown",
			useGPU:         true,
			expectedGPU:    "h264_nvenc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override GPU availability for testing
			originalAvailable := gm.capability.Available
			originalNVENC := gm.capability.NVENCSupported

			gm.mu.Lock()
			gm.capability.Available = tt.useGPU
			gm.capability.NVENCSupported = tt.useGPU
			gm.mu.Unlock()

			codec := gm.GetOptimalCodec(tt.requestedCodec, tt.useGPU)

			if tt.useGPU && codec != tt.expectedGPU {
				t.Errorf("Expected GPU codec %s, got %s", tt.expectedGPU, codec)
			} else if !tt.useGPU && codec != tt.expectedCPU {
				t.Errorf("Expected CPU codec %s, got %s", tt.expectedCPU, codec)
			}

			// Restore original values
			gm.mu.Lock()
			gm.capability.Available = originalAvailable
			gm.capability.NVENCSupported = originalNVENC
			gm.mu.Unlock()
		})
	}
}

func TestGPUManager_BuildGPUArgs(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	tests := []struct {
		name     string
		gpuIndex int
		codec    string
		preset   string
	}{
		{
			name:     "H264 NVENC Medium",
			gpuIndex: 0,
			codec:    "h264_nvenc",
			preset:   "medium",
		},
		{
			name:     "HEVC NVENC Fast",
			gpuIndex: 1,
			codec:    "hevc_nvenc",
			preset:   "fast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := gm.BuildGPUArgs(tt.gpuIndex, tt.codec, tt.preset)

			if len(args) == 0 {
				t.Error("BuildGPUArgs should return non-empty args")
			}

			// Check for required arguments
			hasHwaccel := false
			hasCodec := false
			hasPreset := false

			for i, arg := range args {
				if arg == "-hwaccel" && i+1 < len(args) && args[i+1] == "cuda" {
					hasHwaccel = true
				}
				if arg == "-c:v" && i+1 < len(args) && args[i+1] == tt.codec {
					hasCodec = true
				}
				if arg == "-preset" {
					hasPreset = true
				}
			}

			if !hasHwaccel {
				t.Error("BuildGPUArgs should include -hwaccel cuda")
			}
			if !hasCodec {
				t.Errorf("BuildGPUArgs should include codec %s", tt.codec)
			}
			if !hasPreset {
				t.Error("BuildGPUArgs should include preset")
			}
		})
	}
}

func TestMapPresetToNVENC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ultrafast", "p1"},
		{"superfast", "p2"},
		{"veryfast", "p3"},
		{"faster", "p4"},
		{"fast", "p5"},
		{"medium", "p6"},
		{"slow", "p6"},
		{"slower", "p7"},
		{"veryslow", "p7"},
		{"unknown", "p6"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapPresetToNVENC(tt.input)
			if result != tt.expected {
				t.Errorf("mapPresetToNVENC(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGPUManager_CheckMinMemory(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	// Set up test data
	gm.mu.Lock()
	gm.capability.MemoryFree = []int64{1000, 2000, 500}
	gm.mu.Unlock()

	tests := []struct {
		name      string
		minMemory int64
		expected  bool
	}{
		{
			name:      "Memory available",
			minMemory: 500,
			expected:  true,
		},
		{
			name:      "Exact match",
			minMemory: 1000,
			expected:  true,
		},
		{
			name:      "Memory not available",
			minMemory: 3000,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gm.CheckMinMemory(tt.minMemory)
			if result != tt.expected {
				t.Errorf("CheckMinMemory(%d) = %v, want %v", tt.minMemory, result, tt.expected)
			}
		})
	}
}

func TestGPUManager_SelectBestGPU(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	// Skip this test if nvidia-smi is not available
	ctx := context.Background()
	if _, err := gm.GetMemoryUsage(ctx); err != nil {
		t.Skip("Skipping SelectBestGPU test: nvidia-smi not available")
	}

	gpuIndex, err := gm.SelectBestGPU(ctx)
	if err != nil {
		t.Fatalf("SelectBestGPU failed: %v", err)
	}

	if gpuIndex < 0 {
		t.Error("SelectBestGPU should return non-negative index")
	}

	t.Logf("Selected GPU: %d", gpuIndex)
}

func TestGPUManager_RefreshCapability(t *testing.T) {
	gm := NewGPUManager("ffmpeg")

	cap1 := gm.GetCapability()
	lastChecked1 := cap1.LastChecked

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Refresh
	gm.RefreshCapability()

	cap2 := gm.GetCapability()
	lastChecked2 := cap2.LastChecked

	if !lastChecked2.After(lastChecked1) {
		t.Error("RefreshCapability should update LastChecked time")
	}
}

// Benchmark tests
func BenchmarkGPUManager_GetOptimalCodec(b *testing.B) {
	gm := NewGPUManager("ffmpeg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gm.GetOptimalCodec("h264", true)
	}
}

func BenchmarkGPUManager_BuildGPUArgs(b *testing.B) {
	gm := NewGPUManager("ffmpeg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gm.BuildGPUArgs(0, "h264_nvenc", "medium")
	}
}
