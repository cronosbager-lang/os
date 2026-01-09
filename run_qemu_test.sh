#!/bin/bash
cd /workspaces/os
sudo timeout 60 qemu-system-x86_64 \
  -m 2G \
  -smp 2 \
  -machine q35 \
  -enable-kvm \
  -cpu host \
  -cdrom /workspaces/os/build/output/mixos-1.0.0-x86_64.iso \
  -boot d \
  -nographic \
  -serial mon:stdio \
  -netdev user,id=net0 \
  -device virtio-net-pci,netdev=net0 \
  2>&1 | tee qemu_boot_final.log

echo "QEMU boot test completed"
