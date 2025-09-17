#!/bin/bash

# Script to generate documentation while avoiding kurrentcloud template issues
# This script temporarily removes kurrentcloud resources from provider registration
# during docs generation to prevent terraform-plugin-docs from creating problematic templates

set -e

PROVIDER_FILE="esc/provider.go"
BACKUP_FILE="esc/provider.go.backup"

echo "Creating backup of provider.go..."
cp "$PROVIDER_FILE" "$BACKUP_FILE"

echo "Temporarily commenting out kurrentcloud resources..."
sed -i '' 's/^[[:space:]]*"kurrentcloud_/\/\/ "kurrentcloud_/g' "$PROVIDER_FILE"

# Function to restore provider.go on exit
cleanup() {
    echo "Restoring original provider.go..."
    mv "$BACKUP_FILE" "$PROVIDER_FILE"

    echo "Cleaning up any auto-generated kurrentcloud files..."
    find . -name "*kurrentcloud*" -type f -delete 2>/dev/null || true
}

# Set up cleanup on script exit
trap cleanup EXIT

echo "Generating documentation..."
go generate

echo "Documentation generation completed successfully!"