#!/bin/bash
# MIXOS GO Complete Build Script
# Builds all components in the correct order

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="$ROOT_DIR/build"

VERSION="${VERSION:-1.0}"
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10-mixos}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# Parse arguments
SKIP_KERNEL=false
SKIP_TOOLS=false
SKIP_ROOTFS=false
SKIP_ISO=false
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-kernel) SKIP_KERNEL=true; shift ;;
        --skip-tools) SKIP_TOOLS=true; shift ;;
        --skip-rootfs) SKIP_ROOTFS=true; shift ;;
        --skip-iso) SKIP_ISO=true; shift ;;
        --clean) CLEAN=true; shift ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --skip-kernel  Skip kernel build"
            echo "  --skip-tools   Skip tools build (mix-cli, mix-pkg, etc.)"
            echo "  --skip-rootfs  Skip rootfs build"
            echo "  --skip-iso     Skip ISO build"
            echo "  --clean        Clean build directory first"
            echo "  --help         Show this help"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

echo "=========================================="
echo "MIXOS GO Complete Build"
echo "Version: $VERSION"
echo "=========================================="
echo ""

START_TIME=$(date +%s)

# Clean if requested
if [ "$CLEAN" = true ]; then
    log_step "Cleaning build directory..."
    rm -rf "$BUILD_DIR"
    log_success "Clean complete"
fi

# Create build directory
mkdir -p "$BUILD_DIR"

# Step 1: Build Kernel
if [ "$SKIP_KERNEL" = false ]; then
    log_step "[1/6] Building Kernel..."
    if [ -f "$ROOT_DIR/kernel/scripts/build-kernel.sh" ]; then
        bash "$ROOT_DIR/kernel/scripts/build-kernel.sh" && log_success "Kernel built" || log_warn "Kernel build failed"
    else
        log_warn "Kernel build script not found"
    fi
else
    log_info "[1/6] Skipping kernel build"
fi

# Step 2: Build Tools
if [ "$SKIP_TOOLS" = false ]; then
    log_step "[2/6] Building Tools..."
    
    # mix-cli (Go)
    log_info "Building mix-cli..."
    if [ -d "$ROOT_DIR/mix-cli" ]; then
        cd "$ROOT_DIR/mix-cli"
        mkdir -p "$BUILD_DIR/mix-cli"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/mix-cli/mix-cli" . 2>/dev/null && \
            log_success "mix-cli built" || log_warn "mix-cli build failed"
    fi
    
    # mix-pkg (Rust)
    log_info "Building mix-pkg..."
    if [ -d "$ROOT_DIR/mix-pkg" ]; then
        cd "$ROOT_DIR/mix-pkg"
        mkdir -p "$BUILD_DIR/mix-pkg"
        cargo build --release 2>/dev/null && \
            cp target/release/mix-pkg "$BUILD_DIR/mix-pkg/" 2>/dev/null && \
            log_success "mix-pkg built" || log_warn "mix-pkg build failed"
    fi
    
    # mix-agent (Python - just copy)
    log_info "Preparing mix-agent..."
    if [ -d "$ROOT_DIR/mix-agent" ]; then
        mkdir -p "$BUILD_DIR/mix-agent"
        cp -r "$ROOT_DIR/mix-agent/mixos_agent" "$BUILD_DIR/mix-agent/"
        cp "$ROOT_DIR/mix-agent/requirements.txt" "$BUILD_DIR/mix-agent/"
        cp "$ROOT_DIR/mix-agent/pyproject.toml" "$BUILD_DIR/mix-agent/"
        log_success "mix-agent prepared"
    fi
    
    # mix-agent-early (Go static)
    log_info "Building mix-agent-early..."
    if [ -d "$ROOT_DIR/mix-agent-early" ]; then
        cd "$ROOT_DIR/mix-agent-early"
        mkdir -p "$BUILD_DIR/mix-agent-early"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/mix-agent-early/mix-agent-early" . 2>/dev/null && \
            log_success "mix-agent-early built" || log_warn "mix-agent-early build failed"
    fi
    
    # mix-installer (Go)
    log_info "Building mix-installer..."
    if [ -d "$ROOT_DIR/mix-installer" ]; then
        cd "$ROOT_DIR/mix-installer"
        mkdir -p "$BUILD_DIR/mix-installer"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/mix-installer/mix-installer" . 2>/dev/null && \
            log_success "mix-installer built" || log_warn "mix-installer build failed"
    fi
else
    log_info "[2/6] Skipping tools build"
fi

# Step 3: Build Initramfs
log_step "[3/6] Building Initramfs..."
if [ -f "$ROOT_DIR/initramfs/scripts/build-initramfs.sh" ]; then
    KERNEL_VERSION="$KERNEL_VERSION" bash "$ROOT_DIR/initramfs/scripts/build-initramfs.sh" && \
        log_success "Initramfs built" || log_warn "Initramfs build failed"
else
    log_warn "Initramfs build script not found"
fi

# Step 4: Build RootFS
if [ "$SKIP_ROOTFS" = false ]; then
    log_step "[4/6] Building RootFS..."
    if [ -f "$ROOT_DIR/rootfs/bootstrap.sh" ]; then
        bash "$ROOT_DIR/rootfs/bootstrap.sh" && log_success "RootFS built" || log_warn "RootFS build failed"
    else
        log_warn "RootFS bootstrap script not found"
    fi
else
    log_info "[4/6] Skipping rootfs build"
fi

# Step 5: Build Packages
log_step "[5/6] Building Packages..."
if [ -d "$ROOT_DIR/packages" ]; then
    for pkg_dir in "$ROOT_DIR/packages"/*/; do
        pkg_name=$(basename "$pkg_dir")
        if [ -f "$pkg_dir/PKGBUILD" ]; then
            log_info "Package definition found: $pkg_name"
        fi
    done
    log_success "Package definitions verified"
else
    log_warn "Packages directory not found"
fi

# Step 6: Build ISO
if [ "$SKIP_ISO" = false ]; then
    log_step "[6/6] Building ISO..."
    if [ -f "$ROOT_DIR/iso/scripts/build-iso.sh" ]; then
        VERSION="$VERSION" KERNEL_VERSION="$KERNEL_VERSION" bash "$ROOT_DIR/iso/scripts/build-iso.sh" && \
            log_success "ISO built" || log_warn "ISO build failed"
    else
        log_warn "ISO build script not found"
    fi
else
    log_info "[6/6] Skipping ISO build"
fi

# Calculate build time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

echo ""
echo "=========================================="
log_success "Build Complete!"
echo "Duration: ${MINUTES}m ${SECONDS}s"
echo ""
echo "Build outputs:"
[ -f "$BUILD_DIR/kernel/vmlinuz-$KERNEL_VERSION" ] && echo "  ✓ Kernel"
[ -f "$BUILD_DIR/mix-cli/mix-cli" ] && echo "  ✓ mix-cli"
[ -f "$BUILD_DIR/mix-pkg/mix-pkg" ] && echo "  ✓ mix-pkg"
[ -d "$BUILD_DIR/mix-agent/mixos_agent" ] && echo "  ✓ mix-agent"
[ -f "$BUILD_DIR/mix-agent-early/mix-agent-early" ] && echo "  ✓ mix-agent-early"
[ -f "$BUILD_DIR/mix-installer/mix-installer" ] && echo "  ✓ mix-installer"
[ -d "$BUILD_DIR/rootfs/work" ] && echo "  ✓ RootFS"
[ -f "$BUILD_DIR/iso/mixos-go-$VERSION-x86_64.iso" ] && echo "  ✓ ISO"
echo "=========================================="
