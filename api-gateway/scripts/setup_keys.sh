#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Create keys directory if it doesn't exist
mkdir -p keys

# Copy public key from auth service
print_status "Copying public key from auth service..."
cp ../auth-service/keys/app.rsa.pub keys/public.pem

# Set proper permissions
chmod 644 keys/public.pem

print_status "Public key copied successfully!"
print_status "Public key: keys/public.pem" 