CREATE TABLE IF NOT EXISTS videos (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    duration REAL NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    format TEXT NOT NULL,
    bitrate INTEGER NOT NULL,
    file_size INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    codec TEXT NOT NULL,
    frame_rate REAL NOT NULL,
    aspect_ratio TEXT NOT NULL,
    audio_codec TEXT,
    audio_bitrate INTEGER,
    audio_channels INTEGER,
    content_type TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    file_extension TEXT NOT NULL,
    sanitized_filename TEXT NOT NULL,
    views INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'processing',
    minio_path TEXT NOT NULL,
    hls_path TEXT,
    thumbnail_path TEXT,
    mp4_path TEXT,
    tags TEXT
);

CREATE TABLE IF NOT EXISTS video_views (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    video_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    viewed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (video_id) REFERENCES videos(id),
    UNIQUE(video_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos(user_id);
CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos(created_at);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);
CREATE INDEX IF NOT EXISTS idx_video_views_video_id ON video_views(video_id);
CREATE INDEX IF NOT EXISTS idx_video_views_user_id ON video_views(user_id);