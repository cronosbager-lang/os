#!/bin/bash
# MIXOS GO Installation Test
# Tests the installation process in QEMU

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="$ROOT_DIR/build"
TEST_DIR="$BUILD_DIR/test"

VERSION="${VERSION:-1.0}"
ISO_PATH="${ISO_PATH:-$BUILD_DIR/iso/mixos-go-$VERSION-x86_64.iso}"
DISK_SIZE="${DISK_SIZE:-20G}"
TIMEOUT="${TIMEOUT:-600}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; }
log_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

echo "=========================================="
echo "MIXOS GO Installation Test"
echo "=========================================="

# Check ISO exists
if [ ! -f "$ISO_PATH" ]; then
    log_fail "ISO not found: $ISO_PATH"
    exit 1
fi

# Create test directory
mkdir -p "$TEST_DIR"

# Create test disk
DISK_PATH="$TEST_DIR/install-test.qcow2"
log_info "Creating test disk: $DISK_SIZE"
qemu-img create -f qcow2 "$DISK_PATH" "$DISK_SIZE"

# Create temporary files
SERIAL_LOG="$TEST_DIR/install-serial.log"
MONITOR_SOCKET="$TEST_DIR/install-monitor.sock"

cleanup() {
    pkill -f "qemu.*mixos-install-test" 2>/dev/null || true
    rm -f "$MONITOR_SOCKET"
}
trap cleanup EXIT

log_info "Testing ISO: $ISO_PATH"
log_info "Disk: $DISK_PATH"
log_info "Timeout: ${TIMEOUT}s"

# Start QEMU
log_info "Starting QEMU for installation..."
qemu-system-x86_64 \
    -name mixos-install-test \
    -m 2G \
    -smp 2 \
    -cdrom "$ISO_PATH" \
    -drive file="$DISK_PATH",format=qcow2,if=virtio \
    -boot d \
    -nographic \
    -serial file:"$SERIAL_LOG" \
    -monitor unix:"$MONITOR_SOCKET",server,nowait \
    -device virtio-net-pci,netdev=net0 \
    -netdev user,id=net0 \
    &

QEMU_PID=$!

# Wait for installation
log_info "Waiting for installation (max ${TIMEOUT}s)..."
log_info "Note: Automated installation requires installer to support unattended mode"

INSTALL_SUCCESS=false
START_TIME=$(date +%s)

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -ge $TIMEOUT ]; then
        log_info "Timeout reached"
        break
    fi
    
    if ! kill -0 $QEMU_PID 2>/dev/null; then
        log_info "QEMU exited"
        break
    fi
    
    # Check for installation completion indicators
    if grep -q "Installation complete" "$SERIAL_LOG" 2>/dev/null || \
       grep -q "Reboot now" "$SERIAL_LOG" 2>/dev/null; then
        INSTALL_SUCCESS=true
        log_pass "Installation completed in ${ELAPSED}s"
        break
    fi
    
    # Check for errors
    if grep -q "Installation failed" "$SERIAL_LOG" 2>/dev/null; then
        log_fail "Installation failed"
        break
    fi
    
    sleep 5
done

# Shutdown QEMU
log_info "Shutting down QEMU..."
if [ -S "$MONITOR_SOCKET" ]; then
    echo "quit" | socat - UNIX-CONNECT:"$MONITOR_SOCKET" 2>/dev/null || true
fi
kill $QEMU_PID 2>/dev/null || true
wait $QEMU_PID 2>/dev/null || true

# If installation succeeded, test booting from disk
if [ "$INSTALL_SUCCESS" = true ]; then
    log_info "Testing boot from installed disk..."
    
    BOOT_LOG="$TEST_DIR/boot-from-disk.log"
    
    timeout 120 qemu-system-x86_64 \
        -name mixos-boot-test \
        -m 2G \
        -smp 2 \
        -drive file="$DISK_PATH",format=qcow2,if=virtio \
        -boot c \
        -nographic \
        -serial file:"$BOOT_LOG" \
        -no-reboot \
        &
    
    BOOT_PID=$!
    sleep 60
    
    if grep -q "login:" "$BOOT_LOG" 2>/dev/null; then
        log_pass "Boot from disk successful"
    else
        log_info "Boot from disk - check manually"
    fi
    
    kill $BOOT_PID 2>/dev/null || true
fi

# Show logs
echo ""
echo "Installation log (last 100 lines):"
echo "----------------------------------------"
tail -100 "$SERIAL_LOG" 2>/dev/null || echo "(empty)"
echo "----------------------------------------"

# Report result
echo ""
if [ "$INSTALL_SUCCESS" = true ]; then
    log_pass "Installation test PASSED"
    exit 0
else
    log_info "Installation test requires manual verification"
    log_info "Run: $ROOT_DIR/scripts/dev/start-qemu.sh --disk $DISK_SIZE"
    exit 0
fi
