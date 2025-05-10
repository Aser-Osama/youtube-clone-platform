package validation

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const (
	MaxTitleLength     = 100
	MinTitleLength     = 1
	MaxFileSize        = 1024 * 1024 * 1024 * 5 // 5GB
	MinFileSize        = 1024                   // 1KB
	MaxVideoDuration   = 3600                   // 1 hour in seconds
	MinVideoDuration   = 1                      // 1 second
	MaxVideoResolution = 4320                   // 8K
	MinVideoResolution = 144                    // 144p
)

var (
	ErrTitleTooLong      = errors.New("title is too long")
	ErrTitleTooShort     = errors.New("title is too short")
	ErrFileTooLarge      = errors.New("file is too large")
	ErrFileTooSmall      = errors.New("file is too small")
	ErrInvalidFileType   = errors.New("invalid file type")
	ErrInvalidUserID     = errors.New("invalid user ID format")
	ErrInvalidResolution = errors.New("invalid video resolution")
	ErrInvalidDuration   = errors.New("invalid video duration")
)

// Supported video codecs and their MIME types
var SupportedFormats = map[string][]string{
	"h264": {"video/mp4", "video/quicktime"},
	"vp8":  {"video/webm"},
	"vp9":  {"video/webm"},
	"av1":  {"video/mp4", "video/webm"},
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateTitle checks if the video title is valid
func ValidateTitle(title string) error {
	title = strings.TrimSpace(title)
	if len(title) > MaxTitleLength {
		return &ValidationError{
			Field:   "title",
			Message: fmt.Sprintf("title must be at most %d characters", MaxTitleLength),
		}
	}
	if len(title) < MinTitleLength {
		return &ValidationError{
			Field:   "title",
			Message: fmt.Sprintf("title must be at least %d character", MinTitleLength),
		}
	}
	return nil
}

// ValidateUserID checks if the user ID is valid
func ValidateUserID(userID string) error {
	if userID == "" {
		return &ValidationError{
			Field:   "user_id",
			Message: "user ID is required",
		}
	}
	if !strings.HasPrefix(userID, "test_user_") && !strings.HasPrefix(userID, "google_") {
		return &ValidationError{
			Field:   "user_id",
			Message: "user ID must start with 'test_user_' or 'google_'",
		}
	}
	return nil
}

// ValidateVideoFile checks if the uploaded file is a valid video file
func ValidateVideoFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > MaxFileSize {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file size must be at most %d MB", MaxFileSize/1024/1024),
		}
	}
	if header.Size < MinFileSize {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file size must be at least %d KB", MinFileSize/1024),
		}
	}

	// Use mimetype library to validate file type based on content
	file, err := header.Open()
	if err != nil {
		return &ValidationError{
			Field:   "file",
			Message: "failed to open file",
		}
	}
	defer file.Close()

	mime, err := mimetype.DetectReader(file)
	if err != nil || !strings.HasPrefix(mime.String(), "video/") {
		return &ValidationError{
			Field:   "file",
			Message: "invalid file type",
		}
	}

	// Check file extension and content type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := header.Header.Get("Content-Type")

	validExtensions := map[string]string{
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".webm": "video/webm",
	}

	if mimeType, ok := validExtensions[ext]; !ok {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("unsupported file extension. Supported extensions: %s", strings.Join(getKeys(validExtensions), ", ")),
		}
	} else if contentType != "" && contentType != "application/octet-stream" && contentType != mimeType {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("invalid content type. Expected %s, got %s", mimeType, contentType),
		}
	}

	return nil
}

// ValidateVideoMetadata checks if the video metadata is valid
func ValidateVideoMetadata(duration float64, width, height int, codec string) error {
	// Check duration
	if duration > MaxVideoDuration {
		return &ValidationError{
			Field:   "duration",
			Message: fmt.Sprintf("video duration must be at most %d seconds", MaxVideoDuration),
		}
	}
	if duration < MinVideoDuration {
		return &ValidationError{
			Field:   "duration",
			Message: fmt.Sprintf("video duration must be at least %d second", MinVideoDuration),
		}
	}

	// Check resolution
	maxDimension := max(width, height)
	if maxDimension > MaxVideoResolution {
		return &ValidationError{
			Field:   "resolution",
			Message: fmt.Sprintf("video resolution must be at most %dp", MaxVideoResolution),
		}
	}
	if maxDimension < MinVideoResolution {
		return &ValidationError{
			Field:   "resolution",
			Message: fmt.Sprintf("video resolution must be at least %dp", MinVideoResolution),
		}
	}

	// Check aspect ratio
	if width <= 0 || height <= 0 {
		return &ValidationError{
			Field:   "resolution",
			Message: "invalid video dimensions",
		}
	}

	// Check codec support
	if _, ok := SupportedFormats[codec]; !ok {
		supportedCodecs := make([]string, 0, len(SupportedFormats))
		for codec := range SupportedFormats {
			supportedCodecs = append(supportedCodecs, codec)
		}
		return &ValidationError{
			Field:   "codec",
			Message: fmt.Sprintf("unsupported video codec. Supported codecs: %s", strings.Join(supportedCodecs, ", ")),
		}
	}

	return nil
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
