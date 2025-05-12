// Configure videojs with HLS-specific options
const player = videojs('videoPlayer', {
    fluid: true,
    html5: {
        vhs: {
            overrideNative: true,
            limitRenditionByPlayerDimensions: false,
            smoothQualityChange: true
        }
    },
    controlBar: {
        children: [
            'playToggle',
            'progressControl',
            'volumePanel',
            'qualitySelector', // Add quality selector if plugin is available
            'fullscreenToggle'
        ]
    },
    autoplay: false // Ensure no autoplay
});

// API endpoints
const API_BASE = 'http://localhost:8085/api/v1/streaming';  // Updated to use API gateway port 8085
const METADATA_API = 'http://localhost:8085/api/v1/metadata'; // Updated to use API gateway port 8085

// DOM elements
const videoList = document.getElementById('videoList');
const qualitySelector = document.getElementById('qualitySelector');
const currentQuality = document.getElementById('currentQuality');
const qualityInfo = document.getElementById('qualityInfo');
const radioHLS = document.getElementById('radioHLS');
const radioMP4 = document.getElementById('radioMP4');
const playStreamButton = document.getElementById('playStream');
const refreshVideosButton = document.getElementById('refreshVideos');
const selectedVideoInfo = document.getElementById('selectedVideoInfo');
const selectedVideoTitle = document.getElementById('selectedVideoTitle');
const selectedVideoDescription = document.getElementById('selectedVideoDescription');
const selectedVideoDuration = document.getElementById('selectedVideoDuration');
const selectedVideoResolution = document.getElementById('selectedVideoResolution');
const selectedVideoId = document.getElementById('selectedVideoId');

let currentVideoId = null;
let hlsInstance = null;
let availableQualities = [];
let autoQuality = true;
let selectedQuality = 'auto';
let preparedManifestUrl = null;
let preparedMP4Url = null;
let selectedVideoData = null;

// Show loading indicator in the video list
function showLoading() {
    videoList.innerHTML = `
        <div class="text-center p-4">
            <div class="spinner-border" role="status">
                <span class="visually-hidden">Loading...</span>
            </div>
            <p class="mt-2">Loading videos...</p>
        </div>
    `;
}

// Show error in the video list
function showError(message) {
    videoList.innerHTML = `
        <div class="alert alert-danger m-3" role="alert">
            ${message}
        </div>
    `;
}

// Fetch recent videos from metadata service
async function fetchRecentVideos() {
    showLoading();

    try {
        console.log('Fetching videos from:', `${METADATA_API}/videos`);

        const response = await fetch(`${METADATA_API}/videos`, {
            headers: {
                'Accept': 'application/json'
            },
            mode: 'cors'
        });

        console.log('Response status:', response.status);

        if (!response.ok) {
            throw new Error(`Failed to fetch videos: ${response.status} ${response.statusText}`);
        }

        const text = await response.text();
        console.log('Response text:', text);

        if (!text) {
            throw new Error('Empty response from server');
        }

        try {
            const videos = JSON.parse(text);
            console.log('Parsed videos:', videos);
            displayVideos(videos);
        } catch (parseError) {
            console.error('JSON parse error:', parseError);
            throw new Error(`Failed to parse JSON: ${parseError.message}`);
        }
    } catch (error) {
        console.error('Error fetching videos:', error);
        showError(`Failed to load videos: ${error.message}`);
    }
}

// Display videos in the list
function displayVideos(videos) {
    if (!videos || videos.length === 0) {
        videoList.innerHTML = `
            <div class="no-videos">
                <i class="bi bi-camera-video"></i>
                <h5>No videos available</h5>
                <p>Upload videos to get started.</p>
            </div>
        `;
        return;
    }

    // First generate HTML for all videos
    const videoItems = videos.map(video => {
        // For now just use a fallback thumbnail, we'll load actual thumbnails after rendering
        const fallbackThumbnail = '/static/fallback-thumbnail.png';

        // Format duration
        const formattedDuration = formatDuration(video.duration || 0);

        // Parse and display description properly - handle SQL null string format
        let description = 'No description available.';
        if (video.description) {
            // Handle SQL NullString format: {"String":"","Valid":false}
            if (typeof video.description === 'object') {
                if (video.description.Valid === true) {
                    description = video.description.String || 'No description available.';
                }
            } else {
                description = String(video.description);
            }
        }

        // Truncate description if needed
        const truncatedDescription = description.substring(0, 100) +
            (description.length > 100 ? '...' : '');

        // Store additional data in the video object for later use
        video.formattedDuration = formattedDuration;
        video.description = description;

        return `
            <div class="video-item" data-video-id="${video.id}">
                <div class="video-thumbnail">
                    <img src="${fallbackThumbnail}" alt="${video.title || 'Untitled Video'}" 
                         data-video-id="${video.id}">
                    <div class="duration">${formattedDuration}</div>
                    <div class="play-icon">
                        <i class="bi bi-play-circle-fill"></i>
                    </div>
                </div>
                <div class="video-info">
                    <h6>${video.title || 'Untitled Video'}</h6>
                    <div class="video-metadata">
                        <div class="upload-date">
                            ${formatDate(video.created_at)}
                        </div>
                        <div class="video-id">
                            ID: ${video.id.substring(0, 8)}...
                        </div>
                    </div>
                </div>
            </div>
        `;
    }).join('');

    videoList.innerHTML = videoItems;

    // After rendering, attempt to load thumbnails for each video
    videos.forEach(async (video) => {
        try {
            const imgElement = document.querySelector(`.video-thumbnail img[data-video-id="${video.id}"]`);
            if (imgElement) {
                // Use the loadThumbnail function instead of direct fetch
                const thumbnailUrl = await loadThumbnail(video.id);
                console.log(`Thumbnail URL for video ${video.id}: ${thumbnailUrl}`);

                // Set the thumbnail URL
                imgElement.src = thumbnailUrl;

                // Add error handling for the image
                imgElement.onerror = async function () {
                    console.warn(`Failed to load thumbnail for ${video.id}, retrying...`);
                    // Wait a bit before retrying
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    // Try again with a fresh timestamp
                    const retryUrl = await loadThumbnail(video.id);
                    if (retryUrl !== '/static/fallback-thumbnail.png') {
                        this.src = retryUrl;
                    }
                };
            }
        } catch (error) {
            console.error(`Error loading thumbnail for video ${video.id}:`, error);
        }
    });

    // Add click handlers
    document.querySelectorAll('.video-item').forEach(item => {
        item.addEventListener('click', () => {
            const videoId = item.dataset.videoId;
            selectVideo(videoId, videos.find(v => v.id === videoId));
        });
    });
}

// Format date for display
function formatDate(timestamp) {
    if (!timestamp) return 'Unknown date';

    try {
        const date = new Date(timestamp);
        // If the date is invalid, return a placeholder
        if (isNaN(date.getTime())) return 'Unknown date';

        return date.toLocaleDateString(undefined, {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    } catch (e) {
        return 'Unknown date';
    }
}

// Format duration in seconds to MM:SS
function formatDuration(seconds) {
    if (!seconds || seconds <= 0) return '0:00';

    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
}

// Select a video
function selectVideo(videoId, videoData) {
    if (!videoId) return;

    currentVideoId = videoId;
    selectedVideoData = videoData || null;

    // Update UI selections
    document.querySelectorAll('.video-item').forEach(item => {
        item.classList.toggle('active', item.dataset.videoId === videoId);
    });

    // Reset URLs when a new video is selected
    preparedManifestUrl = null;
    preparedMP4Url = null;

    // Update video info panel
    updateVideoInfoPanel();

    // Enable the quality selector (dropdown) immediately
    qualitySelector.disabled = false;

    // Auto-prepare the video based on selected format
    if (radioHLS.checked) {
        prepareHLS();
    } else if (radioMP4.checked) {
        prepareMP4();
    }
}

// Update video info panel with the selected video's details
function updateVideoInfoPanel() {
    if (!selectedVideoData) {
        selectedVideoInfo.style.display = 'none';
        return;
    }

    // Show the info panel
    selectedVideoInfo.style.display = 'block';

    // Update video details
    selectedVideoTitle.textContent = selectedVideoData.title || 'Untitled Video';
    selectedVideoDescription.textContent = selectedVideoData.description || 'No description available.';
    selectedVideoDuration.textContent = selectedVideoData.formattedDuration || '0:00';
    selectedVideoId.textContent = currentVideoId || '--';

    // Resolution will be updated when playing starts
    selectedVideoResolution.textContent = 'Available after playback';
}

// Reset quality selector
function resetQualitySelector() {
    qualitySelector.innerHTML = '<option value="auto">Auto (Adaptive)</option>';
    qualitySelector.disabled = true;
    qualityInfo.classList.add('d-none');
    currentQuality.textContent = 'Auto';
    availableQualities = [];
    autoQuality = true;
    selectedQuality = 'auto';
}

// Extract available qualities from master playlist
function parseQualities(masterPlaylistContent) {
    // Reset qualities
    resetQualitySelector();

    console.log('Parsing master playlist content:', masterPlaylistContent);

    // Parse the m3u8 content to extract available resolutions
    const lines = masterPlaylistContent.split('\n');
    const qualities = [];

    console.log('Master playlist has', lines.length, 'lines');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        console.log(`Line ${i}: ${line}`);

        // Look for STREAM-INF lines which contain bandwidth and resolution info
        if (line.includes('#EXT-X-STREAM-INF')) {
            const resolutionMatch = line.match(/RESOLUTION=(\d+x\d+)/);
            const bandwidthMatch = line.match(/BANDWIDTH=(\d+)/);
            console.log('Found STREAM-INF line:', line);
            console.log('Resolution match:', resolutionMatch);
            console.log('Bandwidth match:', bandwidthMatch);

            if (resolutionMatch && bandwidthMatch && i + 1 < lines.length) {
                // The URL is on the next line
                const resolution = resolutionMatch[1];
                const bandwidth = parseInt(bandwidthMatch[1], 10);
                const url = lines[i + 1];
                console.log('URL line:', url);

                // Extract quality label from URL which could be either:
                // 1. "/videos/{videoID}/hls/720p/playlist" (full path)
                // 2. "720p/playlist.m3u8" (relative path)
                let quality = null;

                // Try full path pattern first
                const fullPathMatch = url.match(/\/hls\/([^\/]+)\/playlist/);
                if (fullPathMatch) {
                    quality = fullPathMatch[1];
                } else {
                    // Try relative path pattern
                    const relativePathMatch = url.match(/^(\d+p)\/playlist/);
                    if (relativePathMatch) {
                        quality = relativePathMatch[1];
                    } else {
                        // Last attempt - extract from the URL directly
                        // This handles cases like "720p/playlist.m3u8"
                        const directQualityMatch = url.match(/^([^\/]+)\/playlist\.m3u8$/);
                        if (directQualityMatch) {
                            quality = directQualityMatch[1];
                        }
                    }
                }

                console.log('Extracted quality:', quality);

                if (quality) {
                    console.log('Found quality option:', quality, 'with resolution', resolution);
                    qualities.push({
                        quality,
                        resolution,
                        bandwidth,
                        url
                    });
                } else {
                    console.warn('Could not extract quality from URL:', url);
                }
            }
        }
    }

    // Sort by bandwidth (highest first)
    qualities.sort((a, b) => b.bandwidth - a.bandwidth);

    console.log('Available qualities after parsing:', qualities);

    // Update the quality selector
    if (qualities.length > 0) {
        console.log('Adding', qualities.length, 'quality options to selector');
        availableQualities = qualities;

        // Add options to the selector
        qualities.forEach(q => {
            const option = document.createElement('option');
            option.value = q.quality;
            option.textContent = `${q.quality} (${q.resolution})`;
            qualitySelector.appendChild(option);
        });

        qualitySelector.disabled = false;

        // Update the video resolution display
        if (selectedVideoData && qualities.length > 0) {
            const highestQuality = qualities[0];
            selectedVideoResolution.textContent = highestQuality.resolution || highestQuality.quality;
        }
    } else {
        console.warn('No quality options found in manifest');
    }

    return qualities;
}

// Update displayed quality
function updateQualityDisplay(newQuality, isLoading = false) {
    // Clear any previous hide timeout
    if (window.qualityDisplayTimeout) {
        clearTimeout(window.qualityDisplayTimeout);
    }

    // If we're in a loading state, show that instead
    if (isLoading) {
        currentQuality.innerHTML = `${newQuality} <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>`;
    } else {
        currentQuality.textContent = newQuality || 'Auto';
    }

    qualityInfo.classList.remove('d-none');

    // If not loading, hide after delay
    if (!isLoading) {
        window.qualityDisplayTimeout = setTimeout(() => {
            qualityInfo.classList.add('d-none');
        }, 3000);
    }
}

// Prepare HLS stream (fetch manifest but don't play yet)
async function prepareHLS() {
    if (!currentVideoId) {
        alert('Please select a video first');
        return;
    }

    try {
        resetQualitySelector();

        // First, get the manifest content directly (not the redirect)
        const manifestResponse = await fetch(`${API_BASE}/videos/${currentVideoId}/hls/manifest`);
        if (!manifestResponse.ok) {
            throw new Error('Failed to get HLS manifest');
        }

        // Parse the manifest to get quality options
        const manifestContent = await manifestResponse.text();
        const qualities = parseQualities(manifestContent);

        // If no quality options were found, add a note to the UI
        if (qualities.length === 0) {
            console.warn('No quality options found in the HLS manifest');
            const option = document.createElement('option');
            option.value = 'default';
            option.textContent = 'Default (No quality options)';
            qualitySelector.appendChild(option);
        }

        // Get the manifest URL (for compatibility with browsers that don't need HLS.js)
        preparedManifestUrl = `${API_BASE}/videos/${currentVideoId}/hls/manifest`;
        console.log('HLS manifest URL:', preparedManifestUrl);

        // Update the quality display
        updateQualityDisplay(qualities.length === 0 ? "Default" : "Ready - Select Quality");

        // Keep the quality selector enabled but make it read-only if no options
        qualitySelector.disabled = false;
    } catch (error) {
        console.error('Error preparing HLS:', error);
        alert('Failed to prepare HLS stream. Please try again.');
    }
}

// Prepare MP4 (fetch URL but don't play yet)
async function prepareMP4() {
    if (!currentVideoId) {
        alert('Please select a video first');
        return;
    }

    try {
        // Reset quality UI
        resetQualitySelector();

        // First, try to determine available MP4 qualities
        const mp4Qualities = await discoverMP4Qualities(currentVideoId);

        // Set default URL (highest quality)
        preparedMP4Url = `${API_BASE}/videos/${currentVideoId}/mp4`;
        console.log('Default MP4 URL:', preparedMP4Url);

        // Update quality selector with available MP4 qualities
        if (mp4Qualities.length > 0) {
            console.log('Adding MP4 quality options to selector');

            // Add options to the selector
            mp4Qualities.forEach(quality => {
                const option = document.createElement('option');
                option.value = quality.quality;
                option.textContent = `MP4 ${quality.quality}`;
                qualitySelector.appendChild(option);
            });

            // Select the highest quality by default
            if (mp4Qualities.length > 0) {
                selectedQuality = mp4Qualities[0].quality;
                qualitySelector.value = selectedQuality;
                preparedMP4Url = `${API_BASE}/videos/${currentVideoId}/mp4?quality=${selectedQuality}`;

                // Update the video resolution display if we have a selected video
                if (selectedVideoData && mp4Qualities.length > 0) {
                    // For MP4, the quality is typically in the format "720p"
                    const quality = mp4Qualities[0].quality;
                    // Convert to resolution if it's a standard format like "720p"
                    const resMatch = quality.match(/(\d+)p/);
                    if (resMatch) {
                        const height = resMatch[1];
                        // Estimate width based on 16:9 aspect ratio
                        const width = Math.round((parseInt(height) * 16) / 9);
                        selectedVideoResolution.textContent = `${width}x${height}`;
                    } else {
                        selectedVideoResolution.textContent = quality;
                    }
                }
            }
        } else {
            // No specific qualities found, just add a generic MP4 option
            const option = document.createElement('option');
            option.value = "default";
            option.textContent = "MP4 (Default)";
            qualitySelector.appendChild(option);
            selectedQuality = "default";
        }

        // Keep the quality selector enabled
        qualitySelector.disabled = false;

        // Update quality display for MP4
        updateQualityDisplay(selectedQuality === "default" ? "MP4" : `MP4 ${selectedQuality}`);
    } catch (error) {
        console.error('Error preparing MP4:', error);
        alert('Failed to prepare MP4. Please try again.');
    }
}

// Discover available MP4 qualities for a video
async function discoverMP4Qualities(videoId) {
    // Define standard qualities to check - in order of highest to lowest
    const standardQualities = ['1080p', '720p', '480p', '360p', '240p'];
    const availableQualities = [];

    console.log('Discovering MP4 qualities for video ID:', videoId);

    // Check if the video exists first with a regular GET request to the default URL
    try {
        const defaultResponse = await fetch(`${API_BASE}/videos/${videoId}/mp4`, {
            method: 'GET',
            // Don't follow redirects so we can check the status
            redirect: 'manual'
        });

        console.log('Default MP4 response status:', defaultResponse.status);

        // If the default URL works, we know at least some MP4 is available
        if (defaultResponse.status >= 200 && defaultResponse.status < 400) {
            console.log('Default MP4 URL is valid');

            // Try each standard quality with regular GET requests instead of HEAD
            for (const quality of standardQualities) {
                try {
                    console.log(`Checking quality ${quality} for video ${videoId}...`);
                    const response = await fetch(`${API_BASE}/videos/${videoId}/mp4?quality=${quality}`, {
                        method: 'GET',
                        redirect: 'manual' // Don't follow redirects
                    });

                    console.log(`Quality ${quality} response status:`, response.status);

                    // If we get a success status or a redirect (which means the quality exists but redirects)
                    if (response.status >= 200 && response.status < 400 || response.status === 302 || response.status === 307) {
                        console.log(`Found MP4 quality: ${quality}`);
                        availableQualities.push({
                            quality,
                            url: `${API_BASE}/videos/${videoId}/mp4?quality=${quality}`
                        });
                    }
                } catch (error) {
                    console.warn(`Error checking MP4 quality ${quality}:`, error);
                }
            }
        }
    } catch (error) {
        console.warn('Error checking default MP4 URL:', error);
    }

    // If we couldn't verify specific qualities, we'll still have the default as fallback
    if (availableQualities.length === 0) {
        console.log('No specific MP4 qualities found through API checks, using default');

        // Try an alternative approach - check MinIO directly if we have access to the storage service
        try {
            // Make a request to a special endpoint that lists available MP4 qualities
            // You would need to add this endpoint to your backend
            const qualitiesResponse = await fetch(`${API_BASE}/videos/${videoId}/mp4/qualities`);

            if (qualitiesResponse.ok) {
                const qualitiesData = await qualitiesResponse.json();
                console.log('Retrieved MP4 qualities from backend:', qualitiesData);

                if (qualitiesData.qualities && qualitiesData.qualities.length > 0) {
                    for (const quality of qualitiesData.qualities) {
                        availableQualities.push({
                            quality,
                            url: `${API_BASE}/videos/${videoId}/mp4?quality=${quality}`
                        });
                    }
                }
            }
        } catch (error) {
            console.warn('Error fetching MP4 qualities from backend:', error);
        }

        // If we still have no qualities, add the default one
        if (availableQualities.length === 0) {
            // Just add a few standard qualities anyway - they'll fallback to the highest available
            // This ensures the user still gets quality options even if discovery failed
            standardQualities.forEach(quality => {
                availableQualities.push({
                    quality,
                    url: `${API_BASE}/videos/${videoId}/mp4?quality=${quality}`
                });
            });
        }
    }

    console.log('Found MP4 qualities:', availableQualities);
    return availableQualities;
}

// Play HLS stream
function playHLS() {
    if (!preparedManifestUrl) {
        prepareHLS();
        return;
    }

    try {
        // Clean up previous HLS instance if it exists
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }

        if (Hls.isSupported()) {
            hlsInstance = new Hls({
                debug: false,
                // Buffer settings for smoother quality switching
                maxBufferLength: 30,          // Shorter buffer to reduce memory usage but still enough for smooth playback
                maxMaxBufferLength: 60,       // Maximum buffer size
                // Aggressive buffer settings to prevent freezing during quality switches
                highBufferWatchdogPeriod: 2,  // More frequent checking of buffer levels
                // Fragment loading settings
                maxLoadingDelay: 4,           // Tolerate more loading delay
                // Fragment loading tuning
                fragLoadingMaxRetry: 8,       // More retries for fragment loading
                fragLoadingRetryDelay: 500,   // Start retry sooner
                fragLoadingMaxRetryTimeout: 5000, // Cap retry delay
                // Prefetch fragments for smoother switching
                startFragPrefetch: true,
                // Prioritize continuity over quality
                levelLoadingTimeOut: 10000,   // Longer timeout for level loading
                manifestLoadingTimeOut: 10000 // Longer timeout for manifest loading
            });

            // Increase buffer immediately after level switch
            hlsInstance.on(Hls.Events.LEVEL_SWITCHED, (event, data) => {
                // Try to increase buffer after level switch to prevent freezing
                hlsInstance.config.maxBufferLength = 60;

                // After switching level, log the quality change and update UI
                if (autoQuality) {
                    const level = hlsInstance.levels[data.level];
                    if (level) {
                        const height = level.height;
                        const closestQuality = availableQualities.find(q => {
                            const qHeight = parseInt(q.resolution.split('x')[1], 10);
                            return qHeight === height;
                        });
                        updateQualityDisplay(closestQuality ? closestQuality.quality : `${height}p`);

                        // Update video resolution in the details panel
                        if (selectedVideoData) {
                            selectedVideoResolution.textContent = `${level.width}x${level.height}`;
                        }
                    }
                }

                // After a delay, restore normal buffer setting
                setTimeout(() => {
                    hlsInstance.config.maxBufferLength = 30;
                }, 5000);
            });

            // Handle quality switching
            hlsInstance.on(Hls.Events.LEVEL_SWITCHING, (event, data) => {
                console.log(`Switching from level ${data.prevLevel} to ${data.level}`);

                // Try to be more aggressive in loading new fragments
                hlsInstance.config.fragLoadingMaxRetry = 8;
            });

            // Custom error handling
            hlsInstance.on(Hls.Events.ERROR, function (event, data) {
                if (data.fatal) {
                    console.error('Fatal error:', data);
                    switch (data.type) {
                        case Hls.ErrorTypes.NETWORK_ERROR:
                            // Try to recover network error
                            console.log("Fatal network error encountered, trying to recover");
                            hlsInstance.startLoad();
                            break;
                        case Hls.ErrorTypes.MEDIA_ERROR:
                            console.log("Fatal media error encountered, trying to recover");
                            hlsInstance.recoverMediaError();
                            break;
                        default:
                            // Cannot recover
                            hlsInstance.destroy();
                            break;
                    }
                } else {
                    // Log non-fatal errors
                    console.warn('Non-fatal error:', data);
                }
            });

            hlsInstance.loadSource(preparedManifestUrl);
            hlsInstance.attachMedia(player.tech().el());

            hlsInstance.on(Hls.Events.MANIFEST_PARSED, () => {
                console.log('Manifest parsed, found levels:', hlsInstance.levels);

                // Pre-buffer all quality levels for faster switching
                hlsInstance.levels.forEach((level, idx) => {
                    console.log(`Pre-loading level ${idx} (${level.height}p)`);
                });

                // Apply selected quality if needed
                if (selectedQuality !== 'auto') {
                    changeQuality(selectedQuality);
                }

                // Ensure quality selector remains enabled
                qualitySelector.disabled = false;

                // Now we can play
                player.play();
            });
        } else if (player.tech().el().canPlayType('application/vnd.apple.mpegurl')) {
            // For browsers with native HLS support (Safari, iOS)
            player.src({
                src: preparedManifestUrl,
                type: 'application/vnd.apple.mpegurl'
            });

            // Ensure quality selector remains enabled for native HLS playback too
            qualitySelector.disabled = false;

            player.play();
        } else {
            throw new Error('HLS is not supported in this browser');
        }
    } catch (error) {
        console.error('Error playing HLS:', error);
        alert('Failed to play HLS stream. Please try again.');
    }
}

// Play MP4
function playMP4() {
    if (!preparedMP4Url) {
        prepareMP4();
        return;
    }

    try {
        // Clean up HLS instance if it exists
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }

        player.src({
            type: 'video/mp4',
            src: preparedMP4Url
        });

        // Keep quality selector enabled when playing MP4
        qualitySelector.disabled = false;

        player.play().catch(err => {
            console.error('Error playing video:', err);
        });
    } catch (error) {
        console.error('Error playing MP4:', error);
        alert('Failed to play MP4. Please try again.');
    }
}

// Add loading overlay while quality is switching
function addLoadingOverlay() {
    // Remove any existing overlay first
    removeLoadingOverlay();

    // Create the loading overlay
    const overlay = document.createElement('div');
    overlay.id = 'videoLoadingOverlay';
    overlay.style.position = 'absolute';
    overlay.style.top = '0';
    overlay.style.left = '0';
    overlay.style.width = '100%';
    overlay.style.height = '100%';
    overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.7)';
    overlay.style.display = 'flex';
    overlay.style.justifyContent = 'center';
    overlay.style.alignItems = 'center';
    overlay.style.zIndex = '100';

    // Add spinner
    const spinner = document.createElement('div');
    spinner.className = 'spinner-border text-light';
    spinner.setAttribute('role', 'status');
    spinner.style.width = '3rem';
    spinner.style.height = '3rem';

    // Add loading text
    const loadingTextContainer = document.createElement('div');
    loadingTextContainer.style.display = 'flex';
    loadingTextContainer.style.flexDirection = 'column';
    loadingTextContainer.style.alignItems = 'center';

    const loadingText = document.createElement('div');
    loadingText.textContent = 'Loading quality...';
    loadingText.style.color = 'white';
    loadingText.style.marginTop = '10px';
    loadingText.style.fontWeight = 'bold';

    loadingTextContainer.appendChild(spinner);
    loadingTextContainer.appendChild(loadingText);
    overlay.appendChild(loadingTextContainer);

    // Add to player container
    const playerContainer = document.querySelector('.video-container');
    playerContainer.style.position = 'relative'; // Ensure container has positioning
    playerContainer.appendChild(overlay);
}

// Remove loading overlay
function removeLoadingOverlay() {
    const existingOverlay = document.getElementById('videoLoadingOverlay');
    if (existingOverlay) {
        existingOverlay.parentNode.removeChild(existingOverlay);
    }
}

// Change quality level
function changeQuality(quality) {
    if (!hlsInstance) return;

    selectedQuality = quality;
    console.log(`Attempting to change quality to: ${quality}`);

    // Show loading indicator and quality name immediately
    updateQualityDisplay(quality, true);

    // Store current playback position and state
    const currentTime = player.currentTime();
    const wasPlaying = !player.paused();

    // Pause video during quality change to prevent skipping
    if (wasPlaying) {
        player.pause();
    }

    // Add a loading overlay to the video
    addLoadingOverlay();

    if (quality === 'auto') {
        autoQuality = true;
        hlsInstance.nextLevel = -1;    // Set next level to auto for smoother switching
        hlsInstance.loadLevel = -1;    // Also set loadLevel for smoother auto switching
        hlsInstance.currentLevel = -1; // Finally set current level to auto
        console.log('Set to auto quality mode');

        // For auto mode, we can resume immediately
        setTimeout(() => {
            removeLoadingOverlay();
            updateQualityDisplay('Auto');
            if (wasPlaying) player.play();
        }, 500); // Short delay for UI feedback

        return;
    }

    autoQuality = false;
    // Find the level index in Hls.js that matches the selected quality
    const availableQuality = availableQualities.find(q => q.quality === quality);

    if (!availableQuality) {
        console.warn('Selected quality not found in availableQualities:', quality);
        removeLoadingOverlay();
        updateQualityDisplay('Auto (no match found)');
        if (wasPlaying) player.play();
        return;
    }

    console.log('Found matching quality option:', availableQuality);

    const levels = hlsInstance.levels;
    console.log('Available HLS.js levels:', levels);

    // First try to match by resolution height (most reliable)
    const targetHeight = parseInt(availableQuality.resolution.split('x')[1], 10);
    let levelIndex = levels.findIndex(lvl => lvl.height === targetHeight);
    console.log(`Looking for height ${targetHeight}px, found level index: ${levelIndex}`);

    // If that fails, try matching by resolution width
    if (levelIndex === -1) {
        const targetWidth = parseInt(availableQuality.resolution.split('x')[0], 10);
        levelIndex = levels.findIndex(lvl => lvl.width === targetWidth);
        console.log(`Looking for width ${targetWidth}px, found level index: ${levelIndex}`);
    }

    // Finally try matching by bandwidth as a fallback
    if (levelIndex === -1) {
        // Find the level with the closest bandwidth
        let closestBandwidth = Number.MAX_VALUE;
        levels.forEach((level, index) => {
            const bwDiff = Math.abs(level.bitrate - availableQuality.bandwidth);
            if (bwDiff < closestBandwidth) {
                closestBandwidth = bwDiff;
                levelIndex = index;
            }
        });
        console.log(`Using bandwidth matching, found level index: ${levelIndex}`);
    }

    if (levelIndex >= 0) {
        console.log(`Setting HLS.js to level index ${levelIndex}`);

        // Set up buffer events to determine when we can safely resume playback
        const qualitySwitchTimeoutId = setTimeout(() => {
            // Safety fallback - if no event fires after 5 seconds, remove loading and resume
            removeLoadingOverlay();
            updateQualityDisplay(quality);
            if (wasPlaying) {
                player.currentTime(currentTime);
                player.play();
            }
        }, 5000);

        // Monitor fragment loading to detect when the new quality is ready
        const fragmentLoadedHandler = function (event, data) {
            if (data.frag.level === levelIndex) {
                console.log('Fragment loaded at new quality level');
                clearTimeout(qualitySwitchTimeoutId);
                hlsInstance.off(Hls.Events.FRAG_LOADED, fragmentLoadedHandler);

                // Remove loading overlay and resume playback
                removeLoadingOverlay();
                updateQualityDisplay(quality);
                // Restore the time where we were
                player.currentTime(currentTime);
                if (wasPlaying) player.play();

                // Update the video resolution in the details panel
                if (selectedVideoData && levels[levelIndex]) {
                    const level = levels[levelIndex];
                    selectedVideoResolution.textContent = `${level.width}x${level.height}`;
                }
            }
        };

        // Listen for fragment loading at the new level
        hlsInstance.on(Hls.Events.FRAG_LOADED, fragmentLoadedHandler);

        // Set the quality level
        hlsInstance.nextLevel = levelIndex;
        hlsInstance.currentLevel = levelIndex;

        // Increase buffer size temporarily to help with the transition
        hlsInstance.config.maxBufferLength = 60;

        // Restore normal buffer size after transition
        setTimeout(() => {
            hlsInstance.config.maxBufferLength = 30;
        }, 5000);

        console.log(`Quality change initiated at playback time: ${currentTime.toFixed(2)}s`);
    } else {
        console.warn('Could not find any matching HLS.js level for quality:', quality);
        // Fall back to auto as a last resort
        hlsInstance.currentLevel = -1;
        hlsInstance.nextLevel = -1;
        selectedQuality = 'auto';
        autoQuality = true;

        // Remove loading and resume
        removeLoadingOverlay();
        updateQualityDisplay('Auto (no match found)');
        if (wasPlaying) player.play();
    }
}

// Change MP4 quality
function changeMP4Quality(quality) {
    // Store current playback state
    const currentTime = player.currentTime();
    const wasPlaying = !player.paused();

    // Show loading overlay and update quality display
    addLoadingOverlay();
    updateQualityDisplay(`MP4 ${quality}`, true);

    // Pause during quality change
    if (wasPlaying) {
        player.pause();
    }

    selectedQuality = quality;

    // Update the MP4 URL with the new quality
    if (quality === "default") {
        preparedMP4Url = `${API_BASE}/videos/${currentVideoId}/mp4`;
    } else {
        preparedMP4Url = `${API_BASE}/videos/${currentVideoId}/mp4?quality=${quality}`;
    }

    console.log(`Changing MP4 quality to ${quality}, new URL:`, preparedMP4Url);

    // Update the player source
    player.src({
        type: 'video/mp4',
        src: preparedMP4Url
    });

    // When new source is loaded, restore playback position and state
    player.one('loadeddata', () => {
        removeLoadingOverlay();
        updateQualityDisplay(`MP4 ${quality}`);
        player.currentTime(currentTime);
        if (wasPlaying) {
            player.play().catch(err => {
                console.error('Error resuming playback after quality change:', err);
            });
        }

        // Update the video resolution in the details panel
        if (selectedVideoData) {
            // For MP4, we extract resolution from the quality string if possible
            const resMatch = quality.match(/(\d+)p/);
            if (resMatch) {
                const height = resMatch[1];
                // Estimate width based on 16:9 aspect ratio
                const width = Math.round((parseInt(height) * 16) / 9);
                selectedVideoResolution.textContent = `${width}x${height}`;
            } else {
                selectedVideoResolution.textContent = quality;
            }
        }
    });

    // Fallback in case loading takes too long
    setTimeout(() => {
        removeLoadingOverlay();
        updateQualityDisplay(`MP4 ${quality}`);
    }, 5000);
}

// Add logging to thumbnail fetch
async function loadThumbnail(videoId) {
    // Add a timestamp parameter to force a fresh signed URL
    const timestamp = new Date().getTime();
    const thumbnailUrl = `${API_BASE}/videos/${videoId}/thumbnail?t=${timestamp}`;
    console.log(`Attempting to load thumbnail for video ${videoId} from: ${thumbnailUrl}`);

    try {
        // Make a fetch request to check if the thumbnail exists
        const response = await fetch(thumbnailUrl, {
            method: 'GET',
            redirect: 'follow', // Allow redirects to final MinIO URL
            mode: 'cors',      // Explicitly request CORS mode
            credentials: 'same-origin' // Include credentials if needed
        });

        console.log(`Thumbnail response status: ${response.status}`);
        if (response.redirected) {
            console.log(`Thumbnail redirected to: ${response.url}`);
        }

        if (response.ok) {
            // Return the original API endpoint URL instead of the redirected MinIO URL
            // This ensures we always get a fresh signed URL
            return `${API_BASE}/videos/${videoId}/thumbnail?t=${timestamp}`;
        } else {
            console.error(`Failed to load thumbnail: ${response.statusText}`);
            return '/static/fallback-thumbnail.png';
        }
    } catch (error) {
        console.error(`Error loading thumbnail: ${error.message}`);
        // If there's a CORS error, try a different approach
        if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
            console.log('CORS error detected, trying alternative approach');
            // Try to load the image directly by setting the src attribute
            // This will bypass CORS for image loading
            return `${API_BASE}/videos/${videoId}/thumbnail?t=${timestamp}`;
        }
        return '/static/fallback-thumbnail.png';
    }
}

// Event listeners
playStreamButton.addEventListener('click', () => {
    if (radioHLS.checked) {
        playHLS();
    } else if (radioMP4.checked) {
        playMP4();
    }
});

qualitySelector.addEventListener('change', () => {
    const newQuality = qualitySelector.value;
    if (radioHLS.checked) {
        changeQuality(newQuality);
    } else if (radioMP4.checked) {
        changeMP4Quality(newQuality);
    }
});

// Add refresh button event handler
refreshVideosButton.addEventListener('click', () => {
    fetchRecentVideos();
});

// Track quality changes through player events
player.on('playing', () => {
    if (!qualityInfo.classList.contains('d-none')) {
        // Hide quality info after 3 seconds
        setTimeout(() => {
            qualityInfo.classList.add('d-none');
        }, 3000);
    }

    // Show video info panel when playback starts
    if (selectedVideoData) {
        selectedVideoInfo.style.display = 'block';
    }
});

// Initial load
console.log('Starting initial load of videos');
fetchRecentVideos();