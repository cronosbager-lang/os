#!/bin/bash
# MIXOS Initramfs Build Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INITRAMFS_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$INITRAMFS_DIR")"
BUILD_DIR="$ROOT_DIR/build/initramfs"
WORK_DIR="$BUILD_DIR/work"

KERNEL_VERSION="${KERNEL_VERSION:-6.6.10-mixos}"
OUTPUT="$BUILD_DIR/initramfs-$KERNEL_VERSION.img"

echo "=========================================="
echo "MIXOS Initramfs Build"
echo "Kernel: $KERNEL_VERSION"
echo "=========================================="

# Clean and create work directory
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

# Create directory structure
echo "[1/7] Creating directory structure..."
mkdir -p {bin,sbin,usr/bin,usr/sbin,lib,lib64,etc,dev,proc,sys,run,newroot,tmp}

# Install busybox
echo "[2/7] Installing busybox..."
if command -v busybox &> /dev/null; then
    cp "$(which busybox)" bin/busybox
    chmod 755 bin/busybox
    
    # Create symlinks for busybox applets
    for applet in sh ash mount umount switch_root modprobe insmod lsmod \
                  cat echo ls mkdir mknod sleep ln rm cp mv chmod chown \
                  grep sed awk cut head tail sort uniq wc tr \
                  dmesg reboot poweroff halt; do
        ln -sf busybox "bin/$applet"
    done
else
    echo "WARNING: busybox not found, initramfs may not work"
fi

# Copy kernel modules
echo "[3/7] Copying kernel modules..."
MODULES_SRC="$ROOT_DIR/build/kernel/modules/lib/modules/$KERNEL_VERSION"
if [ -d "$MODULES_SRC" ]; then
    mkdir -p "lib/modules/$KERNEL_VERSION"
    
    # Copy essential modules only
    for mod in ahci sd_mod nvme nvme_core ext4 squashfs overlay \
               usb_storage uas virtio_blk virtio_pci virtio_net \
               e1000 e1000e r8169; do
        find "$MODULES_SRC" -name "${mod}.ko*" -exec cp {} "lib/modules/$KERNEL_VERSION/" \; 2>/dev/null || true
    done
    
    # Copy modules.dep and related files
    cp "$MODULES_SRC/modules."* "lib/modules/$KERNEL_VERSION/" 2>/dev/null || true
else
    echo "WARNING: Kernel modules not found at $MODULES_SRC"
fi

# Copy mix-agent-early
echo "[4/7] Installing mix-agent-early..."
if [ -f "$ROOT_DIR/build/mix-agent-early/mix-agent-early" ]; then
    cp "$ROOT_DIR/build/mix-agent-early/mix-agent-early" usr/bin/
    chmod 755 usr/bin/mix-agent-early
else
    echo "WARNING: mix-agent-early not found"
fi

# Copy init script as init.sh
echo "[5/7] Installing init script..."
cp "$INITRAMFS_DIR/init" init.sh
chmod 755 init.sh

# Create minimal C init wrapper
# Kernel expects /init to be ELF binary, not shell script
cat > init.c << 'INIT_C_EOF'
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>

int main(int argc, char *argv[], char *envp[]) {
    pid_t pid = fork();
    if (pid == 0) {
        // Child: execute shell with init.sh
        execve("/bin/sh", (char *const[]){"/bin/sh", "/init.sh", NULL}, envp);
        perror("execve");
        exit(1);
    } else if (pid > 0) {
        // Parent: wait for child
        int status;
        waitpid(pid, &status, 0);
        return WIFEXITED(status) ? WEXITSTATUS(status) : 1;
    } else {
        perror("fork");
        return 1;
    }
}
INIT_C_EOF

# Compile init wrapper
gcc -static -o init init.c 2>/dev/null || {
    # Fallback: create symlink if gcc not available
    ln -sf /bin/busybox init
}

# Ensure `init` is not dynamically linked. If it is, replace with busybox symlink.
if [ -f init ]; then
    if command -v readelf >/dev/null 2>&1; then
        if readelf -l init 2>/dev/null | grep -q 'INTERP'; then
            echo "Compiled init is dynamically linked; using busybox instead"
            rm -f init
            ln -sf /bin/busybox init
        fi
    fi
fi

# Copy configuration files
echo "[6/7] Copying configuration..."
mkdir -p etc
cat > etc/modules.conf << 'EOF'
# Essential modules to load at boot
ahci
sd_mod
nvme
ext4
squashfs
overlay
usb_storage
EOF

# Create initramfs image
echo "[7/7] Creating initramfs image..."
mkdir -p "$(dirname "$OUTPUT")"
find . | cpio -o -H newc 2>/dev/null | gzip -9 > "$OUTPUT"

# Show result
SIZE=$(du -h "$OUTPUT" | cut -f1)
echo "=========================================="
echo "Initramfs build complete!"
echo "Output: $OUTPUT"
echo "Size: $SIZE"
echo "=========================================="
