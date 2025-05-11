package transcoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// QualityLevel represents a video quality level
type QualityLevel struct {
	Name         string
	Width        int
	Height       int
	Bitrate      int
	AudioBitrate int
}

// OutputFormat represents the output format configuration
type OutputFormat struct {
	Format  string
	Quality QualityLevel
}

// Transcoder handles video transcoding operations
type Transcoder interface {
	// TranscodeToHLS transcodes a video to HLS format with multiple quality levels
	TranscodeToHLS(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error
	// TranscodeToMP4 transcodes a video to MP4 format with multiple quality levels
	TranscodeToMP4(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error
	// GenerateThumbnail generates a thumbnail from a video
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error
	// ExtractMetadata extracts metadata from a video file
	ExtractMetadata(ctx context.Context, inputPath string) (map[string]string, error)
}

// FFmpegTranscoder implements the Transcoder interface using FFmpeg
type FFmpegTranscoder struct {
	ffmpegPath      string
	ffmpegThreads   int
	ffmpegPreset    string
	ffmpegCRF       int
	segmentLength   int
	outputFormats   []string
	outputQualities []string
}

// NewTranscoder creates a new FFmpegTranscoder instance
func NewTranscoder(
	ffmpegPath string,
	ffmpegThreads int,
	ffmpegPreset string,
	ffmpegCRF int,
	segmentLength int,
	outputFormats []string,
	outputQualities []string,
	tempDir string,
) (Transcoder, error) {
	return newFFmpegGoImpl(
		ffmpegPath,
		ffmpegThreads,
		ffmpegPreset,
		ffmpegCRF,
		segmentLength,
		outputFormats,
		outputQualities,
		tempDir,
	)
}

// getQualityLevels returns the appropriate quality levels based on input resolution
func (t *FFmpegTranscoder) getQualityLevels(inputWidth, inputHeight int) []QualityLevel {
	// Define all possible quality levels
	allLevels := []QualityLevel{
		{Name: "4k", Width: 3840, Height: 2160, Bitrate: 15000, AudioBitrate: 192},
		{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 5000, AudioBitrate: 192},
		{Name: "720p", Width: 1280, Height: 720, Bitrate: 2800, AudioBitrate: 128},
		{Name: "480p", Width: 854, Height: 480, Bitrate: 1400, AudioBitrate: 128},
		{Name: "360p", Width: 640, Height: 360, Bitrate: 800, AudioBitrate: 96},
		{Name: "240p", Width: 426, Height: 240, Bitrate: 400, AudioBitrate: 64},
	}

	// Filter quality levels based on input resolution
	var levels []QualityLevel
	for _, level := range allLevels {
		if level.Width <= inputWidth && level.Height <= inputHeight {
			levels = append(levels, level)
		}
	}

	// Ensure we have at least 2 quality levels
	if len(levels) < 2 {
		// If input is lower than 240p, add 240p as the lowest quality
		if inputWidth < 426 || inputHeight < 240 {
			levels = append(levels, allLevels[5]) // 240p
		}
		// If input is lower than 360p, add 360p
		if inputWidth < 640 || inputHeight < 360 {
			levels = append(levels, allLevels[4]) // 360p
		}
	}

	return levels
}

// TranscodeToHLS transcodes a video to HLS format with multiple quality levels
func (t *FFmpegTranscoder) TranscodeToHLS(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error {
	qualityLevels := t.getQualityLevels(inputWidth, inputHeight)

	// Create master playlist
	masterPlaylist := "#EXTM3U\n#EXT-X-VERSION:3\n"
	for _, level := range qualityLevels {
		masterPlaylist += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", level.Bitrate*1000, level.Width, level.Height)
		masterPlaylist += fmt.Sprintf("%s/playlist.m3u8\n", level.Name)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "master.m3u8"), []byte(masterPlaylist), 0644); err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}

	// Transcode each quality level
	for _, level := range qualityLevels {
		qualityDir := filepath.Join(outputDir, level.Name)
		if err := os.MkdirAll(qualityDir, 0755); err != nil {
			return fmt.Errorf("failed to create quality directory: %w", err)
		}

		// Create quality-specific playlist
		playlist := fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:0\n", t.segmentLength)
		if err := os.WriteFile(filepath.Join(qualityDir, "playlist.m3u8"), []byte(playlist), 0644); err != nil {
			return fmt.Errorf("failed to create quality playlist: %w", err)
		}

		// Build FFmpeg command for HLS
		args := []string{
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", t.ffmpegPreset,
			"-crf", strconv.Itoa(t.ffmpegCRF),
			"-maxrate", fmt.Sprintf("%dk", level.Bitrate),
			"-bufsize", fmt.Sprintf("%dk", level.Bitrate*2),
			"-vf", fmt.Sprintf("scale=%d:%d", level.Width, level.Height),
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", level.AudioBitrate),
			"-ar", "48000",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", strconv.Itoa(t.segmentLength),
			"-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(qualityDir, "segment_%03d.ts"),
			"-threads", strconv.Itoa(t.ffmpegThreads),
			filepath.Join(qualityDir, "playlist.m3u8"),
		}

		cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
		}
	}

	return nil
}

// TranscodeToMP4 transcodes a video to MP4 format with multiple quality levels
func (t *FFmpegTranscoder) TranscodeToMP4(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error {
	qualityLevels := t.getQualityLevels(inputWidth, inputHeight)

	// Create MP4 directory
	mp4Dir := filepath.Join(outputDir, "mp4")
	if err := os.MkdirAll(mp4Dir, 0755); err != nil {
		return fmt.Errorf("failed to create MP4 directory: %w", err)
	}

	// Transcode each quality level
	for _, level := range qualityLevels {
		outputPath := filepath.Join(mp4Dir, fmt.Sprintf("%s.mp4", level.Name))

		// Build FFmpeg command for MP4
		args := []string{
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", t.ffmpegPreset,
			"-crf", strconv.Itoa(t.ffmpegCRF),
			"-maxrate", fmt.Sprintf("%dk", level.Bitrate),
			"-bufsize", fmt.Sprintf("%dk", level.Bitrate*2),
			"-vf", fmt.Sprintf("scale=%d:%d", level.Width, level.Height),
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", level.AudioBitrate),
			"-ar", "48000",
			"-ac", "2",
			"-movflags", "+faststart",
			"-threads", strconv.Itoa(t.ffmpegThreads),
			outputPath,
		}

		cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
		}
	}

	return nil
}

// GenerateThumbnail generates a thumbnail from a video
func (t *FFmpegTranscoder) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	args := []string{
		"-i", inputPath,
		"-ss", "00:00:05",
		"-vframes", "1",
		"-q:v", "2",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ExtractMetadata extracts metadata from a video file
func (t *FFmpegTranscoder) ExtractMetadata(ctx context.Context, inputPath string) (map[string]string, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w. Output: %s", err, string(output))
	}

	metadata := make(map[string]string)
	metadata["ffprobe_output"] = string(output)

	return metadata, nil
}
