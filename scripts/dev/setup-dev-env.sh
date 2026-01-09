#!/bin/bash
# MIXOS GO Development Environment Setup
# Sets up the development environment for building MIXOS

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "=========================================="
echo "MIXOS GO Development Environment Setup"
echo "=========================================="
echo ""

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    OS=$(uname -s)
fi

log_info "Detected OS: $OS"

# Install dependencies based on OS
install_deps() {
    case $OS in
        ubuntu|debian)
            log_info "Installing dependencies for Debian/Ubuntu..."
            sudo apt-get update
            sudo apt-get install -y \
                build-essential gcc g++ make cmake \
                git curl wget \
                python3 python3-pip python3-venv \
                golang-go \
                busybox cpio \
                kmod \
                rustc cargo \
                flex bison bc libssl-dev libelf-dev \
                xorriso grub-pc-bin grub-efi-amd64-bin mtools \
                squashfs-tools dosfstools \
                qemu-system-x86 qemu-utils \
                syslinux syslinux-utils isolinux \
                libncurses-dev
            ;;
        fedora|rhel|centos)
            log_info "Installing dependencies for Fedora/RHEL..."
            sudo dnf install -y \
                @development-tools gcc gcc-c++ make cmake \
                git curl wget \
                python3 python3-pip \
                golang \
                busybox cpio \
                kmod \
                rust cargo \
                flex bison bc openssl-devel elfutils-libelf-devel \
                xorriso grub2-tools grub2-efi-x64 mtools \
                squashfs-tools dosfstools \
                qemu-system-x86 qemu-img \
                syslinux \
                ncurses-devel
            ;;
        arch|manjaro)
            log_info "Installing dependencies for Arch Linux..."
            sudo pacman -Syu --noconfirm \
                base-devel gcc make cmake \
                git curl wget \
                python python-pip \
                go \
                busybox cpio \
                kmod \
                rust \
                flex bison bc openssl libelf \
                xorriso grub mtools \
                squashfs-tools dosfstools \
                qemu-full \
                syslinux \
                ncurses
            ;;
        *)
            log_warn "Unknown OS: $OS"
            log_info "Please install the following manually:"
            echo "  - GCC, Make, CMake"
            echo "  - Python 3, pip"
            echo "  - Go 1.21+"
            echo "  - Rust, Cargo"
            echo "  - xorriso, grub, mtools"
            echo "  - squashfs-tools, dosfstools"
            echo "  - QEMU"
            ;;
    esac
}

# Setup Go modules
setup_go() {
    log_info "Setting up Go modules..."
    
    for dir in mix-cli mix-installer mix-agent-early; do
        if [ -d "$ROOT_DIR/$dir" ] && [ -f "$ROOT_DIR/$dir/go.mod" ]; then
            log_info "  Downloading dependencies for $dir..."
            cd "$ROOT_DIR/$dir"
            go mod download 2>/dev/null || log_warn "Failed to download Go deps for $dir"
        fi
    done
    
    log_success "Go modules ready"
}

# Setup Rust
setup_rust() {
    log_info "Setting up Rust..."
    
    if [ -d "$ROOT_DIR/mix-pkg" ] && [ -f "$ROOT_DIR/mix-pkg/Cargo.toml" ]; then
        cd "$ROOT_DIR/mix-pkg"
        cargo fetch 2>/dev/null || log_warn "Failed to fetch Rust deps"
    fi
    
    log_success "Rust ready"
}

# Setup Python
setup_python() {
    log_info "Setting up Python environment..."
    
    if [ -d "$ROOT_DIR/mix-agent" ]; then
        cd "$ROOT_DIR/mix-agent"
        
        # Create virtual environment
        if [ ! -d "venv" ]; then
            python3 -m venv venv
        fi
        
        # Install dependencies
        source venv/bin/activate
        pip install --upgrade pip
        pip install -r requirements.txt 2>/dev/null || log_warn "Failed to install Python deps"
        deactivate
    fi
    
    log_success "Python environment ready"
}

# Create build directories
setup_dirs() {
    log_info "Creating build directories..."
    
    mkdir -p "$ROOT_DIR/build"/{kernel,mix-cli,mix-pkg,mix-agent,mix-agent-early,mix-installer,rootfs,initramfs,iso}
    
    log_success "Build directories created"
}

# Main
main() {
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        log_warn "Running as root is not recommended for development"
    fi
    
    # Install system dependencies
    read -p "Install system dependencies? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        install_deps
    fi
    
    # Setup language environments
    setup_dirs
    setup_go
    setup_rust
    setup_python
    
    echo ""
    echo "=========================================="
    log_success "Development environment ready!"
    echo ""
    echo "Next steps:"
    echo "  1. Build everything: make all"
    echo "  2. Build specific component: make mix-cli"
    echo "  3. Test with QEMU: make -C iso test"
    echo "=========================================="
}

main "$@"
