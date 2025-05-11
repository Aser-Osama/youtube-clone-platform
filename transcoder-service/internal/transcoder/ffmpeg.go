package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Quality represents a video quality setting
type Quality struct {
	Name    string
	Width   int
	Height  int
	Bitrate string
}

// ffmpegGoImpl handles video transcoding operations
type ffmpegGoImpl struct {
	ffmpegPath          string
	ffmpegThreads       int
	ffmpegPreset        string
	ffmpegCRF           int
	ffmpegSegmentLength int
	outputFormats       []string
	outputQualities     []Quality
	tempDir             string
}

// newFFmpegGoImpl creates a new transcoder
func newFFmpegGoImpl(ffmpegPath string, ffmpegThreads int, ffmpegPreset string, ffmpegCRF int, ffmpegSegmentLength int, outputFormats []string, outputQualities []string, tempDir string) (*ffmpegGoImpl, error) {
	// Parse quality strings into Quality structs
	qualities := make([]Quality, 0, len(outputQualities))
	for _, q := range outputQualities {
		quality, err := parseQuality(q)
		if err != nil {
			return nil, err
		}
		qualities = append(qualities, quality)
	}

	return &ffmpegGoImpl{
		ffmpegPath:          ffmpegPath,
		ffmpegThreads:       ffmpegThreads,
		ffmpegPreset:        ffmpegPreset,
		ffmpegCRF:           ffmpegCRF,
		ffmpegSegmentLength: ffmpegSegmentLength,
		outputFormats:       outputFormats,
		outputQualities:     qualities,
		tempDir:             tempDir,
	}, nil
}

// parseQuality parses a quality string into a Quality struct
func parseQuality(quality string) (Quality, error) {
	switch quality {
	case "1080p":
		return Quality{Name: "1080p", Width: 1920, Height: 1080, Bitrate: "5000k"}, nil
	case "720p":
		return Quality{Name: "720p", Width: 1280, Height: 720, Bitrate: "2800k"}, nil
	case "480p":
		return Quality{Name: "480p", Width: 854, Height: 480, Bitrate: "1400k"}, nil
	case "360p":
		return Quality{Name: "360p", Width: 640, Height: 360, Bitrate: "800k"}, nil
	case "240p":
		return Quality{Name: "240p", Width: 426, Height: 240, Bitrate: "400k"}, nil
	default:
		return Quality{}, fmt.Errorf("unknown quality: %s", quality)
	}
}

// setupFFmpegLogging creates a log file for FFmpeg output
func setupFFmpegLogging(videoID string, tempDir string) (*os.File, error) {
	// Create logs directory if it doesn't exist
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("ffmpeg_%s_%s.log", videoID, timestamp))

	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return logFile, nil
}

// TranscodeToHLS transcodes a video to HLS format with multiple quality levels
func (t *ffmpegGoImpl) TranscodeToHLS(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error {
	// Extract videoID from inputPath
	videoID := filepath.Base(filepath.Dir(inputPath))

	// Setup logging
	logFile, err := setupFFmpegLogging(videoID, t.tempDir)
	if err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}
	defer logFile.Close()

	// Get appropriate quality levels based on input resolution
	qualityLevels := getQualityLevels(inputWidth, inputHeight)

	log.Printf("Starting HLS transcoding for video %s with %d quality levels", videoID, len(qualityLevels))

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	masterPlaylist, err := os.Create(masterPlaylistPath)
	if err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}
	defer masterPlaylist.Close()

	// Write the master playlist header
	masterPlaylist.WriteString("#EXTM3U\n")
	masterPlaylist.WriteString("#EXT-X-VERSION:3\n")

	// Transcode for each quality
	for i, quality := range qualityLevels {
		log.Printf("Transcoding quality level %d/%d: %s (%dx%d)", i+1, len(qualityLevels), quality.Name, quality.Width, quality.Height)

		// Create a directory for this quality
		qualityDir := filepath.Join(outputDir, quality.Name)
		if err := os.MkdirAll(qualityDir, 0755); err != nil {
			return fmt.Errorf("failed to create quality directory: %w", err)
		}

		// Create a playlist for this quality
		playlistPath := filepath.Join(qualityDir, "playlist.m3u8")
		playlist, err := os.Create(playlistPath)
		if err != nil {
			return fmt.Errorf("failed to create playlist: %w", err)
		}
		defer playlist.Close()

		// Write the playlist header
		playlist.WriteString("#EXTM3U\n")
		playlist.WriteString("#EXT-X-VERSION:3\n")
		playlist.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", t.ffmpegSegmentLength))
		playlist.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

		// Build the FFmpeg command
		args := []string{
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", t.ffmpegPreset,
			"-crf", strconv.Itoa(t.ffmpegCRF),
			"-maxrate", fmt.Sprintf("%dk", quality.Bitrate),
			"-bufsize", fmt.Sprintf("%dk", quality.Bitrate*2),
			"-vf", fmt.Sprintf("scale=%d:%d", quality.Width, quality.Height),
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", quality.AudioBitrate),
			"-ar", "48000",
			"-ac", "2",
			"-f", "hls",
			"-hls_time", strconv.Itoa(t.ffmpegSegmentLength),
			"-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(qualityDir, "segment_%03d.ts"),
			"-hls_flags", "independent_segments",
			"-threads", strconv.Itoa(t.ffmpegThreads),
			"-progress", "pipe:1", // Add progress output
			playlistPath,
		}

		// Run the FFmpeg command
		cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)

		// Create a pipe for progress output
		progressPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create progress pipe: %w", err)
		}

		// Set stderr to the log file
		cmd.Stderr = logFile

		// Start the command
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start ffmpeg: %w", err)
		}

		// Read progress in a goroutine
		go func() {
			scanner := bufio.NewScanner(progressPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "out_time_ms=") {
					timeMs := strings.TrimPrefix(line, "out_time_ms=")
					if ms, err := strconv.ParseInt(timeMs, 10, 64); err == nil {
						duration := time.Duration(ms) * time.Microsecond
						log.Printf("Progress for %s: %v", quality.Name, duration.Round(time.Second))
					}
				}
			}
		}()

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to transcode video: %w", err)
		}

		log.Printf("Completed transcoding for quality level %s", quality.Name)

		// Add this quality to the master playlist
		masterPlaylist.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", quality.Bitrate*1000, quality.Width, quality.Height))
		masterPlaylist.WriteString(fmt.Sprintf("%s/playlist.m3u8\n", quality.Name))
	}

	log.Printf("Completed HLS transcoding for video %s", videoID)
	return nil
}

// TranscodeToMP4 transcodes a video to MP4 format with multiple quality levels
func (t *ffmpegGoImpl) TranscodeToMP4(ctx context.Context, inputPath, outputDir string, inputWidth, inputHeight int) error {
	// Extract videoID from inputPath
	videoID := filepath.Base(filepath.Dir(inputPath))

	// Setup logging
	logFile, err := setupFFmpegLogging(videoID, t.tempDir)
	if err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}
	defer logFile.Close()

	// Get appropriate quality levels based on input resolution
	qualityLevels := getQualityLevels(inputWidth, inputHeight)

	// Create MP4 directory
	mp4Dir := filepath.Join(outputDir, "mp4")
	if err := os.MkdirAll(mp4Dir, 0755); err != nil {
		return fmt.Errorf("failed to create MP4 directory: %w", err)
	}

	// Add progress logging for MP4 transcoding
	for _, quality := range qualityLevels {
		outputPath := filepath.Join(mp4Dir, fmt.Sprintf("%s.mp4", quality.Name))

		// Build FFmpeg command for MP4
		args := []string{
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", t.ffmpegPreset,
			"-crf", strconv.Itoa(t.ffmpegCRF),
			"-maxrate", fmt.Sprintf("%dk", quality.Bitrate),
			"-bufsize", fmt.Sprintf("%dk", quality.Bitrate*2),
			"-vf", fmt.Sprintf("scale=%d:%d", quality.Width, quality.Height),
			"-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", quality.AudioBitrate),
			"-ar", "48000",
			"-ac", "2",
			"-movflags", "+faststart",
			"-threads", strconv.Itoa(t.ffmpegThreads),
			"-progress", "pipe:1", // Add progress output
			outputPath,
		}

		cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)

		// Create a pipe for progress output
		progressPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create progress pipe: %w", err)
		}

		// Set stderr to the log file
		cmd.Stderr = logFile

		// Start the command
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start ffmpeg: %w", err)
		}

		// Read progress in a goroutine
		go func(qualityName string) {
			scanner := bufio.NewScanner(progressPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "out_time_ms=") {
					timeMs := strings.TrimPrefix(line, "out_time_ms=")
					if ms, err := strconv.ParseInt(timeMs, 10, 64); err == nil {
						duration := time.Duration(ms) * time.Microsecond
						log.Printf("Progress for %s: %v", qualityName, duration.Round(time.Second))
					}
				}
			}
		}(quality.Name)

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to transcode video: %w", err)
		}

		log.Printf("Completed transcoding for quality level %s", quality.Name)
	}

	// Log final completion message
	log.Printf("Completed MP4 transcoding for video %s", videoID)

	return nil
}

// getQualityLevels returns the appropriate quality levels based on input resolution
func getQualityLevels(inputWidth, inputHeight int) []QualityLevel {
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

// ExtractMetadata extracts metadata from a video file
func (t *ffmpegGoImpl) ExtractMetadata(ctx context.Context, inputPath string) (map[string]string, error) {
	// Build the FFprobe command
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	// Run the FFprobe command
	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Parse the output
	metadata := make(map[string]string)
	metadata["ffprobe_output"] = string(output)

	return metadata, nil
}

// GenerateThumbnail generates a thumbnail from a video
func (t *ffmpegGoImpl) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	// First get the duration
	durationCmd := exec.CommandContext(ctx, t.ffmpegPath,
		"-i", inputPath,
		"-f", "null",
		"-",
	)
	durationOutput, err := durationCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get video duration: %w\nOutput: %s", err, string(durationOutput))
	}

	// Extract duration from output
	durationStr := string(durationOutput)
	var duration float64
	if _, err := fmt.Sscanf(durationStr, "Duration: %f", &duration); err != nil {
		// If we can't parse duration, use a small offset
		duration = 0.1
	}

	// Calculate seek time (10% of duration, but at least 0.1s and at most 5s)
	seekTime := duration * 0.1
	if seekTime < 0.1 {
		seekTime = 0.1
	} else if seekTime > 5.0 {
		seekTime = 5.0
	}

	// Format seek time as HH:MM:SS.mmm
	seekTimeStr := fmt.Sprintf("%02d:%02d:%02.3f",
		int(seekTime)/3600,
		(int(seekTime)%3600)/60,
		seekTime-float64(int(seekTime)/3600*3600)-float64((int(seekTime)%3600)/60*60),
	)

	args := []string{
		"-i", inputPath,
		"-ss", seekTimeStr,
		"-vframes", "1",
		"-q:v", "2",
		"-vf", "scale=320:-1,format=yuv420p",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
