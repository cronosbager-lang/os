#!/bin/bash
# MIXOS GO QEMU Test Script
# Starts QEMU with the MIXOS ISO for testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="$ROOT_DIR/build"

VERSION="${VERSION:-1.0.0}"
ISO_PATH="${ISO_PATH:-$BUILD_DIR/output/mixos-$VERSION-x86_64.iso}"

# Default settings
MEMORY="${MEMORY:-2G}"
CPUS="${CPUS:-2}"
DISK_SIZE="${DISK_SIZE:-20G}"
UEFI="${UEFI:-false}"
KVM="${KVM:-true}"
GRAPHICS="${GRAPHICS:-true}"
SERIAL="${SERIAL:-false}"
DEBUG="${DEBUG:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -i, --iso PATH      Path to ISO file (default: $ISO_PATH)"
    echo "  -m, --memory SIZE   Memory size (default: $MEMORY)"
    echo "  -c, --cpus NUM      Number of CPUs (default: $CPUS)"
    echo "  -d, --disk SIZE     Create disk of SIZE (default: $DISK_SIZE)"
    echo "  -u, --uefi          Boot in UEFI mode"
    echo "  -n, --no-kvm        Disable KVM acceleration"
    echo "  -s, --serial        Enable serial console"
    echo "  -g, --no-graphics   Disable graphics (serial only)"
    echo "  --debug             Enable QEMU debug options"
    echo "  -h, --help          Show this help"
    echo ""
    echo "Examples:"
    echo "  $0                          # Basic boot"
    echo "  $0 --uefi                   # UEFI boot"
    echo "  $0 --serial --no-graphics   # Serial console only"
    echo "  $0 --disk 50G               # With 50GB disk"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--iso) ISO_PATH="$2"; shift 2 ;;
        -m|--memory) MEMORY="$2"; shift 2 ;;
        -c|--cpus) CPUS="$2"; shift 2 ;;
        -d|--disk) DISK_SIZE="$2"; shift 2 ;;
        -u|--uefi) UEFI=true; shift ;;
        -n|--no-kvm) KVM=false; shift ;;
        -s|--serial) SERIAL=true; shift ;;
        -g|--no-graphics) GRAPHICS=false; shift ;;
        --debug) DEBUG=true; shift ;;
        -h|--help) usage; exit 0 ;;
        *) log_error "Unknown option: $1" ;;
    esac
done

# Check ISO exists
if [ ! -f "$ISO_PATH" ]; then
    log_error "ISO not found: $ISO_PATH"
fi

log_info "Starting QEMU with MIXOS GO"
log_info "ISO: $ISO_PATH"
log_info "Memory: $MEMORY, CPUs: $CPUS"

# Build QEMU command
QEMU_CMD="qemu-system-x86_64"
QEMU_ARGS=()

# Memory and CPU
QEMU_ARGS+=(-m "$MEMORY")
QEMU_ARGS+=(-smp "$CPUS")

# Machine type
QEMU_ARGS+=(-machine q35)

# KVM acceleration
if [ "$KVM" = true ] && [ -e /dev/kvm ]; then
    QEMU_ARGS+=(-enable-kvm)
    QEMU_ARGS+=(-cpu host)
    log_info "KVM acceleration enabled"
else
    QEMU_ARGS+=(-cpu qemu64)
    log_warn "KVM not available, using software emulation"
fi

# UEFI firmware
if [ "$UEFI" = true ]; then
    OVMF_PATHS=(
        "/usr/share/ovmf/OVMF.fd"
        "/usr/share/OVMF/OVMF_CODE.fd"
        "/usr/share/edk2/ovmf/OVMF_CODE.fd"
        "/usr/share/qemu/OVMF.fd"
    )
    
    OVMF_FOUND=""
    for path in "${OVMF_PATHS[@]}"; do
        if [ -f "$path" ]; then
            OVMF_FOUND="$path"
            break
        fi
    done
    
    if [ -n "$OVMF_FOUND" ]; then
        QEMU_ARGS+=(-bios "$OVMF_FOUND")
        log_info "UEFI mode enabled"
    else
        log_warn "OVMF not found, falling back to BIOS mode"
    fi
fi

# Boot from CD-ROM
QEMU_ARGS+=(-cdrom "$ISO_PATH")
QEMU_ARGS+=(-boot d)

# Create and attach disk if requested
if [ -n "$DISK_SIZE" ]; then
    DISK_PATH="$BUILD_DIR/test-disk.qcow2"
    
    if [ ! -f "$DISK_PATH" ]; then
        log_info "Creating test disk: $DISK_SIZE"
        qemu-img create -f qcow2 "$DISK_PATH" "$DISK_SIZE"
    fi
    
    QEMU_ARGS+=(-drive file="$DISK_PATH",format=qcow2,if=virtio)
fi

# Network
QEMU_ARGS+=(-netdev user,id=net0,hostfwd=tcp::2222-:22,hostfwd=tcp::8765-:8765)
QEMU_ARGS+=(-device virtio-net-pci,netdev=net0)

# Graphics
if [ "$GRAPHICS" = true ]; then
    QEMU_ARGS+=(-vga virtio)
    QEMU_ARGS+=(-display gtk)
else
    QEMU_ARGS+=(-nographic)
fi

# Serial console
if [ "$SERIAL" = true ] || [ "$GRAPHICS" = false ]; then
    QEMU_ARGS+=(-serial mon:stdio)
fi

# USB
QEMU_ARGS+=(-usb)
QEMU_ARGS+=(-device usb-tablet)

# Debug options
if [ "$DEBUG" = true ]; then
    QEMU_ARGS+=(-d guest_errors,cpu_reset)
    QEMU_ARGS+=(-D "$BUILD_DIR/qemu-debug.log")
fi

# Show command
echo ""
log_info "QEMU command:"
echo "$QEMU_CMD ${QEMU_ARGS[*]}"
echo ""

# Run QEMU
exec $QEMU_CMD "${QEMU_ARGS[@]}"
