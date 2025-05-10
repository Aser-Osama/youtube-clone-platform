-- name: CreateVideo :exec
INSERT INTO videos (
    id, user_id, title, description, duration, width, height, format,
    bitrate, file_size, checksum, created_at, codec, frame_rate,
    aspect_ratio, audio_codec, audio_bitrate, audio_channels,
    content_type, original_filename, file_extension, sanitized_filename,
    status, minio_path, hls_path, thumbnail_path, tags
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetVideo :one
SELECT * FROM videos WHERE id = ? LIMIT 1;

-- name: GetRecentVideos :many
SELECT * FROM videos 
WHERE status = 'ready'
ORDER BY created_at DESC
LIMIT ?;

-- name: UpdateVideoStatus :exec
UPDATE videos SET status = ? WHERE id = ?;

-- name: IncrementViews :exec
UPDATE videos SET views = views + 1 WHERE id = ?;

-- name: CreateVideoView :exec
INSERT INTO video_views (video_id, user_id) VALUES (?, ?);

-- name: CheckVideoView :one
SELECT COUNT(*) as count FROM video_views WHERE video_id = ? AND user_id = ?;

-- name: SearchVideos :many
SELECT * FROM videos 
WHERE status = 'ready'
AND (
    title LIKE ? OR 
    description LIKE ? OR 
    tags LIKE ?
)
ORDER BY created_at DESC
LIMIT ?;

-- name: GetVideosByUser :many
SELECT * FROM videos 
WHERE user_id = ? AND status = 'ready'
ORDER BY created_at DESC
LIMIT ?; 