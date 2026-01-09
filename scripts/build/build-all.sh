#!/bin/bash
# ============================================================================
# MIXOS GO - Complete Build Script
# ============================================================================
#
# This script builds all components using the Makefile build system.
# It's a convenience wrapper that provides additional options.
#
# Usage:
#   ./build-all.sh [options]
#
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()    { echo -e "${CYAN}[STEP]${NC} $1"; }
log_phase()   { echo -e "\n${WHITE}════════════════════════════════════════════════════════════${NC}\n${WHITE} $1${NC}\n${WHITE}════════════════════════════════════════════════════════════${NC}\n"; }

# Parse arguments
CLEAN=false
CHECK_ONLY=false
TARGET="all"
VERBOSE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean) CLEAN=true; shift ;;
        --check) CHECK_ONLY=true; shift ;;
        --verbose|-v) VERBOSE="V=1"; shift ;;
        --kernel) TARGET="kernel"; shift ;;
        --packages) TARGET="packages"; shift ;;
        --initramfs) TARGET="initramfs"; shift ;;
        --rootfs) TARGET="rootfs"; shift ;;
        --iso) TARGET="iso"; shift ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --clean        Clean build directory first"
            echo "  --check        Only check dependencies"
            echo "  --verbose, -v  Show verbose output"
            echo "  --kernel       Build only kernel"
            echo "  --packages     Build only packages"
            echo "  --initramfs    Build only initramfs"
            echo "  --rootfs       Build only rootfs"
            echo "  --iso          Build only ISO"
            echo "  --help, -h     Show this help"
            exit 0
            ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# Show banner
echo ""
echo -e "${CYAN}"
echo "  ███╗   ███╗██╗██╗  ██╗ ██████╗ ███████╗     ██████╗  ██████╗ "
echo "  ████╗ ████║██║╚██╗██╔╝██╔═══██╗██╔════╝    ██╔════╝ ██╔═══██╗"
echo "  ██╔████╔██║██║ ╚███╔╝ ██║   ██║███████╗    ██║  ███╗██║   ██║"
echo "  ██║╚██╔╝██║██║ ██╔██╗ ██║   ██║╚════██║    ██║   ██║██║   ██║"
echo "  ██║ ╚═╝ ██║██║██╔╝ ██╗╚██████╔╝███████║    ╚██████╔╝╚██████╔╝"
echo "  ╚═╝     ╚═╝╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝     ╚═════╝  ╚═════╝ "
echo -e "${NC}"
echo "                    Complete Build System"
echo ""

cd "$ROOT_DIR"

# Check dependencies
log_phase "Checking Dependencies"
if [ -f "$SCRIPT_DIR/check-deps.sh" ]; then
    bash "$SCRIPT_DIR/check-deps.sh" || {
        log_error "Dependency check failed"
        exit 1
    }
else
    log_warn "Dependency check script not found, skipping"
fi

if [ "$CHECK_ONLY" = true ]; then
    log_success "Dependency check complete"
    exit 0
fi

# Clean if requested
if [ "$CLEAN" = true ]; then
    log_phase "Cleaning Build Directory"
    make clean
    log_success "Clean complete"
fi

# Build
log_phase "Starting Build: $TARGET"

START_TIME=$(date +%s)

# Run make with the specified target
make $VERBOSE $TARGET

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

# Show summary
log_phase "Build Complete"

echo " Build time: ${MINUTES}m ${SECONDS}s"
echo ""

# Show status
make status

echo ""
echo " To test with QEMU:"
echo "   make test-qemu"
echo ""
echo " Or manually:"
echo "   ./scripts/dev/start-qemu.sh build/output/mixos-*.iso"
echo ""
