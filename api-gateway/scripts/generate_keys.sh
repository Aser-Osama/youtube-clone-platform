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

# Generate private key
print_status "Generating private key..."
openssl genrsa -out keys/private.pem 2048

# Generate public key from private key
print_status "Generating public key..."
openssl rsa -in keys/private.pem -pubout -out keys/public.pem

# Set proper permissions
chmod 600 keys/private.pem
chmod 644 keys/public.pem

print_status "Keys generated successfully!"
print_status "Private key: keys/private.pem"
print_status "Public key: keys/public.pem" 