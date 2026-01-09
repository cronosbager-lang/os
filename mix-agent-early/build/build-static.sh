#!/bin/bash
# Build static binary for mix-agent-early

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/../build/mix-agent-early"

echo "Building mix-agent-early (static binary)..."

cd "$PROJECT_DIR"

# Build static binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o "$BUILD_DIR/mix-agent-early" \
    .

# Strip binary
strip "$BUILD_DIR/mix-agent-early" 2>/dev/null || true

# Show size
ls -lh "$BUILD_DIR/mix-agent-early"

echo "Build complete: $BUILD_DIR/mix-agent-early"
