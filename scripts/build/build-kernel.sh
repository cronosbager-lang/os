#!/bin/bash
# ============================================================================
# MIXOS GO - Kernel Build Script
# ============================================================================
#
# This script builds the Linux kernel with MIXOS configuration.
# It uses variables from config.mk (passed via environment).
#
# Outputs:
#   - $KERNEL_IMAGE (vmlinuz-X.X.X-mixos)
#   - $KERNEL_MODULES_DIR (lib/modules/X.X.X-mixos/)
#   - $KERNEL_SYSTEM_MAP (System.map-X.X.X-mixos)
#
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Determine ROOT_DIR (scripts/build -> scripts -> root)
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Kernel source directory
KERNEL_SRC_DIR="${ROOT_DIR}/kernel"

# Set defaults (always, to ensure all variables are set)
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10}"
KERNEL_LOCALVERSION="${KERNEL_LOCALVERSION:--mixos}"
KERNEL_FULL_VERSION="${KERNEL_FULL_VERSION:-${KERNEL_VERSION}${KERNEL_LOCALVERSION}}"
KERNEL_CONFIG="${KERNEL_CONFIG:-mixos-default}"
BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"
KERNEL_BUILD_DIR="${KERNEL_BUILD_DIR:-$BUILD_DIR/kernel}"
CACHE_DIR="${CACHE_DIR:-$BUILD_DIR/cache}"

# Derived paths
KERNEL_TARBALL="linux-${KERNEL_VERSION}.tar.xz"
KERNEL_SOURCE_DIR="${KERNEL_SRC_DIR}/linux-${KERNEL_VERSION}"
KERNEL_MAJOR=$(echo "$KERNEL_VERSION" | cut -d. -f1)
KERNEL_URL="https://cdn.kernel.org/pub/linux/kernel/v${KERNEL_MAJOR}.x/${KERNEL_TARBALL}"

# Output paths (canonical)
KERNEL_IMAGE="${KERNEL_BUILD_DIR}/vmlinuz-${KERNEL_FULL_VERSION}"
KERNEL_MODULES_DIR="${KERNEL_BUILD_DIR}/modules/lib/modules/${KERNEL_FULL_VERSION}"
KERNEL_SYSTEM_MAP="${KERNEL_BUILD_DIR}/System.map-${KERNEL_FULL_VERSION}"

# Build settings
NPROC=$(nproc)
MAKE_FLAGS="-j${NPROC}"

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
# FUNCTIONS
# ============================================================================

check_dependencies() {
    log_step "Checking build dependencies..."
    
    local deps=(gcc make bc flex bison libelf-dev libssl-dev)
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &>/dev/null && ! dpkg -l "$dep" &>/dev/null 2>&1; then
            # Check if it's a library
            if [[ "$dep" == lib* ]]; then
                if ! ldconfig -p | grep -q "$dep"; then
                    missing+=("$dep")
                fi
            else
                missing+=("$dep")
            fi
        fi
    done
    
    if [ ${#missing[@]} -gt 0 ]; then
        log_warn "Some dependencies may be missing: ${missing[*]}"
        log_info "Install with: sudo apt install ${missing[*]}"
    fi
}

download_kernel() {
    log_step "[1/6] Downloading kernel source..."
    
    mkdir -p "$CACHE_DIR/downloads"
    local cached_tarball="$CACHE_DIR/downloads/$KERNEL_TARBALL"
    
    if [ -f "$cached_tarball" ]; then
        log_info "Using cached kernel source: $cached_tarball"
        if [ ! -f "$KERNEL_SRC_DIR/$KERNEL_TARBALL" ]; then
            ln -sf "$cached_tarball" "$KERNEL_SRC_DIR/$KERNEL_TARBALL"
        fi
    elif [ -f "$KERNEL_SRC_DIR/$KERNEL_TARBALL" ]; then
        log_info "Kernel source already downloaded"
    else
        log_info "Downloading from $KERNEL_URL"
        curl -L --progress-bar -o "$cached_tarball" "$KERNEL_URL"
        ln -sf "$cached_tarball" "$KERNEL_SRC_DIR/$KERNEL_TARBALL"
    fi
    
    log_success "Kernel source ready"
}

extract_kernel() {
    log_step "[2/6] Extracting kernel source..."
    
    if [ -d "$KERNEL_SOURCE_DIR" ]; then
        log_info "Kernel source already extracted"
    else
        cd "$KERNEL_SRC_DIR"
        tar xf "$KERNEL_TARBALL"
        log_success "Extracted to $KERNEL_SOURCE_DIR"
    fi
}

apply_patches() {
    log_step "[3/6] Applying patches..."
    
    local patches_dir="$KERNEL_SRC_DIR/patches"
    
    if [ -d "$patches_dir" ] && [ "$(ls -A "$patches_dir"/*.patch 2>/dev/null)" ]; then
        cd "$KERNEL_SOURCE_DIR"
        
        for patch in "$patches_dir"/*.patch; do
            local patch_name=$(basename "$patch")
            
            # Check if patch already applied
            if patch -p1 --dry-run < "$patch" &>/dev/null; then
                log_info "Applying: $patch_name"
                patch -p1 < "$patch"
            else
                log_info "Patch already applied or not applicable: $patch_name"
            fi
        done
        
        log_success "Patches applied"
    else
        log_info "No patches to apply"
    fi
}

configure_kernel() {
    log_step "[4/6] Configuring kernel..."
    
    cd "$KERNEL_SOURCE_DIR"
    
    local config_file="$KERNEL_SRC_DIR/config/${KERNEL_CONFIG}.config"
    
    if [ ! -f "$config_file" ]; then
        log_error "Config file not found: $config_file"
        exit 1
    fi
    
    # Copy config
    cp "$config_file" .config
    
    # Set local version
    echo "$KERNEL_LOCALVERSION" > localversion
    # Remove any existing localversion files to avoid double suffix
    # The CONFIG_LOCALVERSION in .config already has -mixos
    rm -f localversion localversion-*
    # Update config with defaults for new options
    make olddefconfig
    
    # Verify critical options
    log_info "Verifying critical kernel options..."
    
    local critical_options=(
        "CONFIG_MODULES=y"
        "CONFIG_BLK_DEV_INITRD=y"
        "CONFIG_SQUASHFS=y"
        "CONFIG_OVERLAY_FS=y"
        "CONFIG_EXT4_FS=y"
        "CONFIG_DEVTMPFS=y"
        "CONFIG_DEVTMPFS_MOUNT=y"
    )
    
    for opt in "${critical_options[@]}"; do
        local key="${opt%%=*}"
        if ! grep -q "^${key}=" .config; then
            log_warn "Missing critical option: $key"
        fi
    done
    
    log_success "Kernel configured"
}

build_kernel() {
    log_step "[5/6] Building kernel..."
    
    cd "$KERNEL_SOURCE_DIR"
    
    log_info "Building kernel image (bzImage)..."
    make $MAKE_FLAGS bzImage
    
    log_info "Building kernel modules..."
    make $MAKE_FLAGS modules
    
    log_success "Kernel build complete"
}

install_kernel() {
    log_step "[6/6] Installing kernel to build directory..."
    
    cd "$KERNEL_SOURCE_DIR"
    
    # Create output directories
    mkdir -p "$KERNEL_BUILD_DIR"
    mkdir -p "$(dirname "$KERNEL_MODULES_DIR")"
    
    # Install kernel image
    log_info "Installing kernel image..."
    cp arch/x86/boot/bzImage "$KERNEL_IMAGE"
    
    # Install System.map
    log_info "Installing System.map..."
    cp System.map "$KERNEL_SYSTEM_MAP"
    
    # Install modules
    log_info "Installing kernel modules..."
    make INSTALL_MOD_PATH="$KERNEL_BUILD_DIR/modules" modules_install
    
    # Remove build/source symlinks (not needed and cause issues)
    rm -f "$KERNEL_MODULES_DIR/build" "$KERNEL_MODULES_DIR/source"
    
    # Generate module dependencies
    log_info "Generating module dependencies..."
    depmod -b "$KERNEL_BUILD_DIR/modules" "$KERNEL_FULL_VERSION"
    
    log_success "Kernel installed"
}

show_summary() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " Kernel Build Complete"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Version:     $KERNEL_FULL_VERSION"
    echo " Kernel:      $KERNEL_IMAGE"
    echo " System.map:  $KERNEL_SYSTEM_MAP"
    echo " Modules:     $KERNEL_MODULES_DIR"
    echo ""
    
    # Show sizes
    if [ -f "$KERNEL_IMAGE" ]; then
        local kernel_size=$(du -h "$KERNEL_IMAGE" | cut -f1)
        echo " Kernel size: $kernel_size"
    fi
    
    if [ -d "$KERNEL_MODULES_DIR" ]; then
        local modules_size=$(du -sh "$KERNEL_MODULES_DIR" | cut -f1)
        local modules_count=$(find "$KERNEL_MODULES_DIR" -name "*.ko*" | wc -l)
        echo " Modules:     $modules_count modules ($modules_size)"
    fi
    
    echo ""
    echo "════════════════════════════════════════════════════════════════"
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " MIXOS GO - Kernel Build"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Kernel Version:  $KERNEL_VERSION"
    echo " Local Version:   $KERNEL_LOCALVERSION"
    echo " Full Version:    $KERNEL_FULL_VERSION"
    echo " Config:          $KERNEL_CONFIG"
    echo " Build Directory: $KERNEL_BUILD_DIR"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    
    check_dependencies
    download_kernel
    extract_kernel
    apply_patches
    configure_kernel
    build_kernel
    install_kernel
    show_summary
}

# Run main
main "$@"
