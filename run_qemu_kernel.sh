#!/bin/bash
# Simple QEMU kernel boot test dengan output capture

echo "=== QEMU Kernel Boot Test ==="
echo "Start time: $(date)"
echo ""

# Run QEMU dan capture output dengan stderr redirect
# Create temporary disk for testing
TEMP_DISK="/tmp/mixos_test_disk.img"
if [ ! -f "$TEMP_DISK" ]; then
    echo "Creating temporary test disk..."
    dd if=/dev/zero of="$TEMP_DISK" bs=1M count=200 2>/dev/null || true
    
    # Create raw disk image without filesystem for testing
    echo "Creating raw disk image (no filesystem)..."
    # Don't format - just create empty disk
fi

sudo qemu-system-x86_64 \
  -m 1G \
  -smp 2 \
  -enable-kvm \
  -kernel /workspaces/os/build/kernel/vmlinuz-6.6.10-mixos \
  -initrd /workspaces/os/build/initramfs/initramfs-6.6.10-mixos.img \
  -drive file="$TEMP_DISK",if=virtio,format=raw \
  -append "console=ttyS0,115200 root=/dev/vda init=/init" \
  -nographic \
  -serial mon:stdio

echo ""
echo "End time: $(date)"
echo "Exit code: $?"
echo ""
echo "=== Log output (last 200 lines) ==="
tail -200 /workspaces/os/qemu_direct_output.log
