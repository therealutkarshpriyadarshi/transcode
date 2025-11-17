package transcoder

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GPUCapability represents GPU encoding capabilities
type GPUCapability struct {
	Available       bool
	DeviceCount     int
	DeviceNames     []string
	NVENCSupported  bool
	MaxEncoders     int
	MemoryTotal     []int64 // MB per device
	MemoryFree      []int64 // MB per device
	DriverVersion   string
	CUDAVersion     string
	LastChecked     time.Time
}

// GPUManager manages GPU resources and capabilities
type GPUManager struct {
	capability *GPUCapability
	mu         sync.RWMutex
	ffmpegPath string
}

// NewGPUManager creates a new GPU manager
func NewGPUManager(ffmpegPath string) *GPUManager {
	gm := &GPUManager{
		ffmpegPath: ffmpegPath,
		capability: &GPUCapability{},
	}
	gm.detectGPU()
	return gm
}

// detectGPU detects GPU capabilities
func (gm *GPUManager) detectGPU() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cap := &GPUCapability{
		Available:   false,
		LastChecked: time.Now(),
	}

	// Check if nvidia-smi is available
	if err := gm.checkNVIDIASMI(ctx, cap); err != nil {
		gm.capability = cap
		return
	}

	// Check FFmpeg NVENC support
	if err := gm.checkFFmpegNVENC(ctx); err != nil {
		cap.NVENCSupported = false
	} else {
		cap.NVENCSupported = true
		cap.Available = true
	}

	gm.capability = cap
}

// checkNVIDIASMI checks NVIDIA GPU using nvidia-smi
func (gm *GPUManager) checkNVIDIASMI(ctx context.Context, cap *GPUCapability) error {
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name,memory.total,memory.free,driver_version", "--format=csv,noheader")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nvidia-smi not available: %w", err)
	}

	// Parse nvidia-smi output
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	cap.DeviceCount = len(lines)
	cap.DeviceNames = make([]string, 0, cap.DeviceCount)
	cap.MemoryTotal = make([]int64, 0, cap.DeviceCount)
	cap.MemoryFree = make([]int64, 0, cap.DeviceCount)

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 4 {
			// Device name
			name := strings.TrimSpace(parts[0])
			cap.DeviceNames = append(cap.DeviceNames, name)

			// Memory total (convert MiB to MB)
			memTotal := strings.TrimSpace(strings.ReplaceAll(parts[1], " MiB", ""))
			if mt, err := strconv.ParseInt(memTotal, 10, 64); err == nil {
				cap.MemoryTotal = append(cap.MemoryTotal, mt)
			}

			// Memory free
			memFree := strings.TrimSpace(strings.ReplaceAll(parts[2], " MiB", ""))
			if mf, err := strconv.ParseInt(memFree, 10, 64); err == nil {
				cap.MemoryFree = append(cap.MemoryFree, mf)
			}

			// Driver version
			if cap.DriverVersion == "" {
				cap.DriverVersion = strings.TrimSpace(parts[3])
			}
		}
	}

	// Get CUDA version
	cmd = exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=cuda_version", "--format=csv,noheader")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	stdout.Reset()
	stderr.Reset()

	if err := cmd.Run(); err == nil {
		cap.CUDAVersion = strings.TrimSpace(stdout.String())
	}

	// Estimate max encoders (typically 2-3 per GPU)
	cap.MaxEncoders = cap.DeviceCount * 3

	return nil
}

// checkFFmpegNVENC checks if FFmpeg supports NVENC
func (gm *GPUManager) checkFFmpegNVENC(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, gm.ffmpegPath, "-hide_banner", "-encoders")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to check ffmpeg encoders: %w", err)
	}

	output := stdout.String()

	// Check for NVENC encoders
	if !strings.Contains(output, "h264_nvenc") || !strings.Contains(output, "hevc_nvenc") {
		return fmt.Errorf("NVENC encoders not found in FFmpeg")
	}

	return nil
}

// GetCapability returns current GPU capability
func (gm *GPUManager) GetCapability() *GPUCapability {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Return a copy to avoid race conditions
	cap := *gm.capability
	return &cap
}

// IsAvailable checks if GPU encoding is available
func (gm *GPUManager) IsAvailable() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return gm.capability.Available && gm.capability.NVENCSupported
}

// RefreshCapability refreshes GPU capability information
func (gm *GPUManager) RefreshCapability() {
	gm.detectGPU()
}

// GetMemoryUsage returns current GPU memory usage
func (gm *GPUManager) GetMemoryUsage(ctx context.Context) ([]GPUMemoryInfo, error) {
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=index,memory.used,memory.free,memory.total,utilization.gpu", "--format=csv,noheader,nounits")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to query GPU memory: %w", err)
	}

	var memInfo []GPUMemoryInfo
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			index, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			used, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			free, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
			total, _ := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
			util, _ := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)

			memInfo = append(memInfo, GPUMemoryInfo{
				DeviceIndex:     index,
				MemoryUsed:      used,
				MemoryFree:      free,
				MemoryTotal:     total,
				GPUUtilization:  util,
			})
		}
	}

	return memInfo, nil
}

// GPUMemoryInfo holds GPU memory information
type GPUMemoryInfo struct {
	DeviceIndex    int
	MemoryUsed     int64   // MB
	MemoryFree     int64   // MB
	MemoryTotal    int64   // MB
	GPUUtilization float64 // Percentage
}

// SelectBestGPU selects the GPU with most free memory
func (gm *GPUManager) SelectBestGPU(ctx context.Context) (int, error) {
	memInfo, err := gm.GetMemoryUsage(ctx)
	if err != nil {
		return -1, err
	}

	if len(memInfo) == 0 {
		return -1, fmt.Errorf("no GPUs available")
	}

	// Select GPU with most free memory and lowest utilization
	bestIndex := 0
	maxScore := float64(0)

	for _, info := range memInfo {
		// Score based on free memory and inverse utilization
		score := float64(info.MemoryFree) * (100.0 - info.GPUUtilization)
		if score > maxScore {
			maxScore = score
			bestIndex = info.DeviceIndex
		}
	}

	return bestIndex, nil
}

// GetOptimalCodec returns the optimal codec based on GPU capability
func (gm *GPUManager) GetOptimalCodec(requestedCodec string, useGPU bool) string {
	if !useGPU || !gm.IsAvailable() {
		// CPU codecs
		switch requestedCodec {
		case "h264", "libx264", "h264_nvenc":
			return "libx264"
		case "h265", "hevc", "libx265", "hevc_nvenc":
			return "libx265"
		case "vp9", "libvpx-vp9":
			return "libvpx-vp9"
		default:
			return "libx264"
		}
	}

	// GPU codecs
	switch requestedCodec {
	case "h264", "libx264", "h264_nvenc":
		return "h264_nvenc"
	case "h265", "hevc", "libx265", "hevc_nvenc":
		return "hevc_nvenc"
	default:
		return "h264_nvenc"
	}
}

// BuildGPUArgs builds FFmpeg arguments for GPU encoding
func (gm *GPUManager) BuildGPUArgs(gpuIndex int, codec string, preset string) []string {
	args := []string{}

	// Select GPU device
	if gpuIndex >= 0 {
		args = append(args, "-hwaccel", "cuda")
		args = append(args, "-hwaccel_device", fmt.Sprintf("%d", gpuIndex))
	}

	// GPU-specific encoding options
	switch codec {
	case "h264_nvenc":
		args = append(args, "-c:v", "h264_nvenc")

		// NVENC preset mapping
		nvencPreset := mapPresetToNVENC(preset)
		args = append(args, "-preset", nvencPreset)

		// Quality settings
		args = append(args, "-rc", "vbr") // Variable bitrate
		args = append(args, "-cq", "23")  // Constant quality
		args = append(args, "-b:v", "0")  // Let CQ control bitrate
		args = append(args, "-maxrate:v", "0")

		// B-frames
		args = append(args, "-bf", "3")
		args = append(args, "-b_ref_mode", "middle")

		// Profile and level
		args = append(args, "-profile:v", "high")
		args = append(args, "-level", "4.1")

	case "hevc_nvenc":
		args = append(args, "-c:v", "hevc_nvenc")

		nvencPreset := mapPresetToNVENC(preset)
		args = append(args, "-preset", nvencPreset)

		// Quality settings
		args = append(args, "-rc", "vbr")
		args = append(args, "-cq", "23")
		args = append(args, "-b:v", "0")
		args = append(args, "-maxrate:v", "0")

		// B-frames
		args = append(args, "-bf", "4")

		// Profile
		args = append(args, "-profile:v", "main")
		args = append(args, "-tier", "high")
	}

	return args
}

// mapPresetToNVENC maps standard presets to NVENC presets
func mapPresetToNVENC(preset string) string {
	mapping := map[string]string{
		"ultrafast": "p1",
		"superfast": "p2",
		"veryfast":  "p3",
		"faster":    "p4",
		"fast":      "p5",
		"medium":    "p6",
		"slow":      "p6",
		"slower":    "p7",
		"veryslow":  "p7",
	}

	if nvencPreset, ok := mapping[preset]; ok {
		return nvencPreset
	}

	return "p6" // Default to medium quality
}

// CheckMinMemory checks if GPU has minimum required memory
func (gm *GPUManager) CheckMinMemory(minMemoryMB int64) bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for _, memFree := range gm.capability.MemoryFree {
		if memFree >= minMemoryMB {
			return true
		}
	}

	return false
}

// GetEncoderStats returns statistics about NVENC encoder usage
func (gm *GPUManager) GetEncoderStats(ctx context.Context) (map[int]int, error) {
	// Query nvidia-smi for encoder utilization
	cmd := exec.CommandContext(ctx, "nvidia-smi", "dmon", "-s", "u", "-c", "1")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to query encoder stats: %w", err)
	}

	// Parse output to get encoder utilization per GPU
	stats := make(map[int]int)
	lines := strings.Split(stdout.String(), "\n")

	// nvidia-smi dmon output format: # gpu   enc   dec
	re := regexp.MustCompile(`\s*(\d+)\s+(\d+)\s+(\d+)`)

	for _, line := range lines {
		if matches := re.FindStringSubmatch(line); len(matches) == 4 {
			gpu, _ := strconv.Atoi(matches[1])
			enc, _ := strconv.Atoi(matches[2])
			stats[gpu] = enc
		}
	}

	return stats, nil
}
