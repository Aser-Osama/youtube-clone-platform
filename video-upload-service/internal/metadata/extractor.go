package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type VideoMetadata struct {
	Duration          float64 `json:"duration"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	Format            string  `json:"format"`
	Bitrate           int64   `json:"bitrate"`
	FileSize          int64   `json:"file_size"`
	Checksum          string  `json:"checksum"`
	CreatedAt         string  `json:"created_at"`
	Codec             string  `json:"codec"`
	FrameRate         float64 `json:"frame_rate"`
	AspectRatio       string  `json:"aspect_ratio"`
	AudioCodec        string  `json:"audio_codec"`
	AudioBitrate      int64   `json:"audio_bitrate"`
	AudioChannels     int     `json:"audio_channels"`
	ContentType       string  `json:"content_type"`
	OriginalFilename  string  `json:"original_filename"`
	FileExtension     string  `json:"file_extension"`
	SanitizedFilename string  `json:"sanitized_filename"`
}

type ffprobeFormat struct {
	Duration       string            `json:"duration"`
	Filename       string            `json:"filename"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	Size           string            `json:"size"`
	BitRate        string            `json:"bit_rate"`
	Tags           map[string]string `json:"tags"`
}

type ffprobeStream struct {
	CodecType          string `json:"codec_type"`
	CodecName          string `json:"codec_name"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	RFrameRate         string `json:"r_frame_rate"`
	Channels           int    `json:"channels"`
	BitRate            string `json:"bit_rate"`
	SampleRate         string `json:"sample_rate"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
	Profile            string `json:"profile"`
	Level              int    `json:"level"`
	PixFmt             string `json:"pix_fmt"`
	ColorSpace         string `json:"color_space"`
	ColorRange         string `json:"color_range"`
	Index              int    `json:"index"`
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

func runCommandWithTimeout(parentCtx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// ExtractMetadata extracts metadata from a video file using ffprobe with retries
func ExtractMetadata(ctx context.Context, filePath string) (*VideoMetadata, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		metadata, err := extractMetadataOnce(ctx, filePath)
		if err == nil {
			return metadata, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func extractMetadataOnce(ctx context.Context, filePath string) (*VideoMetadata, error) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	// Get original filename from filePath
	originalFilename := filepath.Base(filePath)
	fileExtension := filepath.Ext(originalFilename)
	sanitizedFilename := SanitizeFilename(originalFilename)

	output, err := runCommandWithTimeout(ctx, 5*time.Second,
		"ffprobe", "-v", "error", "-print_format", "json", "-show_format", "-show_streams", filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to execute ffprobe: %w", err)
	}

	// // Debug: Print raw ffprobe output
	// fmt.Printf("Raw ffprobe output:\n%s\n", string(output))

	var ffprobe ffprobeOutput
	if err := json.Unmarshal(output, &ffprobe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// // Debug: Print parsed streams
	// fmt.Printf("Found %d streams:\n", len(ffprobe.Streams))
	// for i, stream := range ffprobe.Streams {
	// 	fmt.Printf("Stream %d: type=%s codec=%s width=%d height=%d\n",
	// 		i, stream.CodecType, stream.CodecName, stream.Width, stream.Height)
	// }

	// Find video and audio streams
	var videoStream *ffprobeStream
	var audioStream *ffprobeStream
	for i := range ffprobe.Streams {
		stream := ffprobe.Streams[i]
		if stream.CodecType == "video" && videoStream == nil {
			videoStream = &stream
		} else if stream.CodecType == "audio" && audioStream == nil {
			audioStream = &stream
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found in file")
	}

	// Parse duration
	duration, err := strconv.ParseFloat(ffprobe.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	// Parse bitrate
	bitrate, err := strconv.ParseInt(ffprobe.Format.BitRate, 10, 64)
	if err != nil {
		bitrate = 0 // Some files might not have bitrate info
	}

	// Parse file size
	size, err := strconv.ParseInt(ffprobe.Format.Size, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid file size: %w", err)
	}

	// Calculate frame rate
	frameRate := 0.0
	if videoStream.RFrameRate != "" {
		parts := strings.Split(videoStream.RFrameRate, "/")
		if len(parts) == 2 {
			num, _ := strconv.ParseFloat(parts[0], 64)
			den, _ := strconv.ParseFloat(parts[1], 64)
			if den != 0 {
				frameRate = num / den
			}
		}
	}

	// Get aspect ratio
	aspectRatio := videoStream.DisplayAspectRatio
	if aspectRatio == "" {
		if videoStream.Width > 0 && videoStream.Height > 0 {
			gcd := greatestCommonDivisor(videoStream.Width, videoStream.Height)
			aspectRatio = fmt.Sprintf("%d:%d", videoStream.Width/gcd, videoStream.Height/gcd)
		} else {
			aspectRatio = "unknown"
		}
	}

	// Calculate checksum (MD5)
	checksumOutput, err := runCommandWithTimeout(ctx, 3*time.Second, "md5sum", filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := strings.Fields(string(checksumOutput))[0]

	// Parse audio metadata
	audioBitrate := int64(0)
	audioChannels := 0
	audioCodec := ""
	if audioStream != nil {
		if bitrate, err := strconv.ParseInt(audioStream.BitRate, 10, 64); err == nil {
			audioBitrate = bitrate
		}
		audioChannels = audioStream.Channels
		audioCodec = audioStream.CodecName
	}

	// Determine content type based on format and codec
	contentType := determineContentType(ffprobe.Format.FormatName, videoStream.CodecName)

	return &VideoMetadata{
		Duration:          duration,
		Width:             videoStream.Width,
		Height:            videoStream.Height,
		Format:            ffprobe.Format.FormatName,
		Bitrate:           bitrate,
		FileSize:          size,
		Checksum:          checksum,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		Codec:             videoStream.CodecName,
		FrameRate:         frameRate,
		AspectRatio:       aspectRatio,
		AudioCodec:        audioCodec,
		AudioBitrate:      audioBitrate,
		AudioChannels:     audioChannels,
		ContentType:       contentType,
		OriginalFilename:  originalFilename,
		FileExtension:     fileExtension,
		SanitizedFilename: sanitizedFilename,
	}, nil
}

func determineContentType(format, codec string) string {
	format = strings.ToLower(format)
	codec = strings.ToLower(codec)

	// Common format mappings
	formatMap := map[string]string{
		"mp4":      "video/mp4",
		"mov":      "video/quicktime",
		"webm":     "video/webm",
		"matroska": "video/x-matroska",
		"avi":      "video/x-msvideo",
	}

	// Check format first
	for key, contentType := range formatMap {
		if strings.Contains(format, key) {
			return contentType
		}
	}

	// Fallback based on codec - always prefer MP4 container for common codecs
	switch codec {
	case "h264", "hevc", "h265", "mpeg4", "avc", "aac":
		return "video/mp4"
	case "vp8", "vp9":
		return "video/webm"
	case "av1":
		return "video/mp4" // Prefer MP4 for AV1 as it's more widely supported
	default:
		// If we can't determine, set to MP4 for better compatibility
		if strings.Contains(format, "iso") || strings.Contains(format, "mp4") {
			return "video/mp4"
		}
		return "video/mp4" // Change default to video/mp4 instead of application/octet-stream
	}
}

func greatestCommonDivisor(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// SanitizeFilename removes or replaces characters that could cause issues in filenames
func SanitizeFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	// Remove any non-alphanumeric characters except underscores and hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	name = reg.ReplaceAllString(name, "")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Add timestamp to ensure uniqueness
	timestamp := time.Now().Format("20060102_150405")

	return fmt.Sprintf("%s_%s", name, timestamp)
}
