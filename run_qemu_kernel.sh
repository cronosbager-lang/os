#!/bin/bash
# Simple QEMU kernel boot test dengan output capture

echo "=== QEMU Kernel Boot Test ==="
echo "Start time: $(date)"
echo ""

# Run QEMU dan capture output dengan stderr redirect
sudo timeout 45 qemu-system-x86_64 \
  -m 1G \
  -smp 2 \
  -enable-kvm \
  -kernel /workspaces/os/build/kernel/vmlinuz-6.6.10-mixos \
  -initrd /workspaces/os/build/initramfs/initramfs-6.6.10-mixos.img \
  -append "console=ttyS0,115200 init=/bin/sh" \
  -nographic \
  2>&1 > /workspaces/os/qemu_direct_output.log

echo ""
echo "End time: $(date)"
echo "Exit code: $?"
echo ""
echo "=== Log output (last 200 lines) ==="
tail -200 /workspaces/os/qemu_direct_output.log
