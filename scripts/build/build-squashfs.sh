#!/bin/bash
# ============================================================================
# MIXOS GO - Squashfs Build Script
# ============================================================================
#
# This script compresses the root filesystem to a squashfs image.
# The squashfs image is used for live boot from ISO.
#
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source config if not already set
if [ -z "${BUILD_DIR:-}" ]; then
    BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"
    ROOTFS_BUILD_DIR="${ROOTFS_BUILD_DIR:-$BUILD_DIR/rootfs}"
fi

# Paths
ROOTFS_ROOT="${ROOTFS_BUILD_DIR}/root"
ROOTFS_SQUASHFS="${ROOTFS_BUILD_DIR}/rootfs.sfs"

# Compression settings
SQUASHFS_COMPRESSION="${SQUASHFS_COMPRESSION:-zstd}"
SQUASHFS_COMP_LEVEL="${SQUASHFS_COMP_LEVEL:-19}"
SQUASHFS_BLOCK_SIZE="${SQUASHFS_BLOCK_SIZE:-1M}"

# ============================================================================
# COLORS
# ============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()    { echo -e "${CYAN}[STEP]${NC} $1"; }

# ============================================================================
# MAIN
# ============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " MIXOS GO - Squashfs Build"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    
    # Check prerequisites
    if [ ! -d "$ROOTFS_ROOT" ]; then
        log_error "Rootfs not found: $ROOTFS_ROOT"
        log_error "Run 'make rootfs' first"
        exit 1
    fi
    
    if ! command -v mksquashfs &>/dev/null; then
        log_error "mksquashfs not found"
        log_error "Install squashfs-tools package"
        exit 1
    fi
    
    # Show source info
    local src_size=$(du -sh "$ROOTFS_ROOT" | cut -f1)
    log_info "Source:      $ROOTFS_ROOT ($src_size)"
    log_info "Output:      $ROOTFS_SQUASHFS"
    log_info "Compression: $SQUASHFS_COMPRESSION (level $SQUASHFS_COMP_LEVEL)"
    log_info "Block size:  $SQUASHFS_BLOCK_SIZE"
    echo ""
    
    # Remove old squashfs if exists
    rm -f "$ROOTFS_SQUASHFS"
    
    # Create squashfs
    log_step "Creating squashfs image..."
    
    local mksquashfs_opts=(
        "$ROOTFS_ROOT"
        "$ROOTFS_SQUASHFS"
        -comp "$SQUASHFS_COMPRESSION"
        -b "$SQUASHFS_BLOCK_SIZE"
        -no-exports
        -no-recovery
        -always-use-fragments
    )
    
    # Add compression level for supported compressors
    case "$SQUASHFS_COMPRESSION" in
        zstd)
            mksquashfs_opts+=(-Xcompression-level "$SQUASHFS_COMP_LEVEL")
            ;;
        xz)
            mksquashfs_opts+=(-Xbcj x86)
            ;;
        gzip)
            mksquashfs_opts+=(-Xcompression-level "$SQUASHFS_COMP_LEVEL")
            ;;
    esac
    
    mksquashfs "${mksquashfs_opts[@]}"
    
    # Show result
    echo ""
    if [ -f "$ROOTFS_SQUASHFS" ]; then
        local out_size=$(du -h "$ROOTFS_SQUASHFS" | cut -f1)
        local out_bytes=$(stat -c%s "$ROOTFS_SQUASHFS")
        local src_bytes=$(du -sb "$ROOTFS_ROOT" | cut -f1)
        local ratio=$((100 - (out_bytes * 100 / src_bytes)))
        
        echo "════════════════════════════════════════════════════════════════"
        echo " Squashfs Build Complete"
        echo "════════════════════════════════════════════════════════════════"
        echo ""
        echo " Output:           $ROOTFS_SQUASHFS"
        echo " Original size:    $src_size"
        echo " Compressed size:  $out_size"
        echo " Compression:      ${ratio}% reduction"
        echo ""
        echo "════════════════════════════════════════════════════════════════"
        
        log_success "Squashfs created successfully"
    else
        log_error "Failed to create squashfs"
        exit 1
    fi
}

# Run main
main "$@"
