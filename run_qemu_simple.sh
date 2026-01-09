#!/bin/bash
echo "Starting QEMU boot test..."
cd /workspaces/os

# Try with KVM first, fallback to software emulation if fails
timeout 90 qemu-system-x86_64 \
  -m 1G \
  -smp 2 \
  -machine q35 \
  -enable-kvm \
  -cpu host \
  -cdrom /workspaces/os/build/output/mixos-1.0.0-x86_64.iso \
  -boot d \
  -nographic \
  -serial mon:stdio \
  2>&1

echo ""
echo "QEMU boot test finished at $(date)"
