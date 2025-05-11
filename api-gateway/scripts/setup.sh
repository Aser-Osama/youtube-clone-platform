#!/bin/bash

# Create .env file
cat > .env << EOL
# Server configuration
SERVER_PORT=8085

# Service URLs
AUTH_SERVICE_URL=http://localhost:8080
METADATA_SERVICE_URL=http://localhost:8082
STREAMING_SERVICE_URL=http://localhost:8083
UPLOAD_SERVICE_URL=http://localhost:8084

# JWT configuration
JWT_PUBLIC_KEY_PATH=keys/public.pem

# Rate limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_PERIOD=1m
EOL

# Create keys directory if it doesn't exist
mkdir -p keys

echo "Environment setup complete. Edit .env if needed." 