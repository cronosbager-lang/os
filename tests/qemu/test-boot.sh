#!/bin/bash
# MIXOS GO Boot Test
# Tests that the ISO boots successfully in QEMU

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="$ROOT_DIR/build"

VERSION="${VERSION:-1.0.0}"
ISO_PATH="${ISO_PATH:-$BUILD_DIR/output/mixos-$VERSION-x86_64.iso}"
TIMEOUT="${TIMEOUT:-120}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; }
log_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

echo "=========================================="
echo "MIXOS GO Boot Test"
echo "=========================================="

# Check ISO exists
if [ ! -f "$ISO_PATH" ]; then
    log_fail "ISO not found: $ISO_PATH"
    exit 1
fi

log_info "Testing ISO: $ISO_PATH"
log_info "Timeout: ${TIMEOUT}s"

# Create temporary files
SERIAL_LOG=$(mktemp)
MONITOR_SOCKET=$(mktemp -u)

cleanup() {
    rm -f "$SERIAL_LOG" "$MONITOR_SOCKET"
    # Kill any remaining QEMU processes
    pkill -f "qemu.*mixos-test" 2>/dev/null || true
}
trap cleanup EXIT

# Start QEMU in background
log_info "Starting QEMU..."
qemu-system-x86_64 \
    -name mixos-test \
    -m 1G \
    -smp 2 \
    -cdrom "$ISO_PATH" \
    -boot d \
    -nographic \
    -serial file:"$SERIAL_LOG" \
    -monitor unix:"$MONITOR_SOCKET",server,nowait \
    -no-reboot \
    -device virtio-net-pci,netdev=net0 \
    -netdev user,id=net0 \
    &

QEMU_PID=$!

# Wait for boot with timeout
log_info "Waiting for boot (max ${TIMEOUT}s)..."

BOOT_SUCCESS=false
START_TIME=$(date +%s)

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -ge $TIMEOUT ]; then
        log_fail "Boot timeout after ${TIMEOUT}s"
        break
    fi
    
    # Check if QEMU is still running
    if ! kill -0 $QEMU_PID 2>/dev/null; then
        log_fail "QEMU exited unexpectedly"
        break
    fi
    
    # Check serial log for boot indicators
    if grep -q "MIXOS GO" "$SERIAL_LOG" 2>/dev/null; then
        log_info "Found MIXOS GO banner"
    fi
    
    if grep -q "login:" "$SERIAL_LOG" 2>/dev/null || \
       grep -q "Welcome to MIXOS" "$SERIAL_LOG" 2>/dev/null || \
       grep -q "systemd.*Reached target" "$SERIAL_LOG" 2>/dev/null; then
        BOOT_SUCCESS=true
        log_pass "System booted successfully in ${ELAPSED}s"
        break
    fi
    
    # Check for kernel panic
    if grep -q "Kernel panic" "$SERIAL_LOG" 2>/dev/null; then
        log_fail "Kernel panic detected"
        break
    fi
    
    sleep 2
done

# Shutdown QEMU
log_info "Shutting down QEMU..."
if [ -S "$MONITOR_SOCKET" ]; then
    echo "quit" | socat - UNIX-CONNECT:"$MONITOR_SOCKET" 2>/dev/null || true
fi
kill $QEMU_PID 2>/dev/null || true
wait $QEMU_PID 2>/dev/null || true

# Show serial log excerpt
echo ""
echo "Serial log (last 50 lines):"
echo "----------------------------------------"
tail -50 "$SERIAL_LOG" 2>/dev/null || echo "(empty)"
echo "----------------------------------------"

# Report result
echo ""
if [ "$BOOT_SUCCESS" = true ]; then
    log_pass "Boot test PASSED"
    exit 0
else
    log_fail "Boot test FAILED"
    exit 1
fi
