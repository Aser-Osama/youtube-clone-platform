package types

// VideoUploadEvent represents a video upload event from the upload service
type VideoUploadEvent struct {
	VideoID     string        `json:"video_id"`
	UserID      string        `json:"user_id"`
	Title       string        `json:"title"`
	ContentType string        `json:"content_type"`
	Size        int64         `json:"size"`
	Metadata    VideoMetadata `json:"metadata"`
	UploadedAt  string        `json:"uploaded_at"`
}

// VideoMetadata represents the metadata extracted from a video file
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
