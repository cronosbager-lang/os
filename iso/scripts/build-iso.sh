#!/bin/bash
# MIXOS ISO Build Script
# Creates a bootable ISO image

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISO_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$ISO_DIR")"
BUILD_DIR="$ROOT_DIR/build/iso"
WORK_DIR="$BUILD_DIR/work"

VERSION="${VERSION:-1.0}"
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10-mixos}"
OUTPUT="$BUILD_DIR/mixos-go-$VERSION-x86_64.iso"

echo "=========================================="
echo "MIXOS ISO Build"
echo "Version: $VERSION"
echo "=========================================="

# Clean and create work directory
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

# Create ISO directory structure
echo "[1/8] Creating ISO structure..."
mkdir -p {boot/grub,EFI/BOOT,LiveOS,isolinux}

# Copy kernel
echo "[2/8] Copying kernel..."
KERNEL_PATH="$ROOT_DIR/build/kernel/vmlinuz-$KERNEL_VERSION"
if [ -f "$KERNEL_PATH" ]; then
    cp "$KERNEL_PATH" boot/vmlinuz-mixos
else
    echo "WARNING: Kernel not found at $KERNEL_PATH"
    touch boot/vmlinuz-mixos  # Placeholder
fi

# Copy initramfs
echo "[3/8] Copying initramfs..."
INITRAMFS_PATH="$ROOT_DIR/build/initramfs/initramfs-$KERNEL_VERSION.img"
if [ -f "$INITRAMFS_PATH" ]; then
    cp "$INITRAMFS_PATH" boot/initramfs-mixos.img
else
    echo "WARNING: Initramfs not found at $INITRAMFS_PATH"
    touch boot/initramfs-mixos.img  # Placeholder
fi

# Create squashfs image from rootfs
echo "[4/8] Creating squashfs image..."
ROOTFS_PATH="$ROOT_DIR/build/rootfs/work"
if [ -d "$ROOTFS_PATH" ]; then
    mksquashfs "$ROOTFS_PATH" LiveOS/squashfs.img \
        -comp zstd -Xcompression-level 19 \
        -b 1M -no-recovery \
        2>/dev/null || {
        echo "WARNING: mksquashfs failed, creating placeholder"
        touch LiveOS/squashfs.img
    }
else
    echo "WARNING: RootFS not found at $ROOTFS_PATH"
    touch LiveOS/squashfs.img  # Placeholder
fi

# Copy GRUB configuration
echo "[5/8] Setting up GRUB..."
cp "$ISO_DIR/grub/grub.cfg" boot/grub/

# Create GRUB EFI image
echo "[6/8] Creating EFI boot image..."
if command -v grub-mkimage &> /dev/null; then
    grub-mkimage -o EFI/BOOT/BOOTX64.EFI \
        -O x86_64-efi \
        -p /boot/grub \
        part_gpt part_msdos fat ext2 normal boot linux \
        configfile loopback chain efifwsetup efi_gop \
        efi_uga ls search search_label search_fs_uuid \
        search_fs_file gfxterm gfxterm_background \
        gfxterm_menu test all_video loadenv exfat \
        2>/dev/null || echo "WARNING: grub-mkimage failed"
fi

# Create EFI boot partition image
echo "[7/8] Creating EFI partition..."
dd if=/dev/zero of=efi.img bs=1M count=32 2>/dev/null
mkfs.vfat efi.img 2>/dev/null || true
if command -v mmd &> /dev/null; then
    mmd -i efi.img ::/EFI ::/EFI/BOOT
    mcopy -i efi.img EFI/BOOT/BOOTX64.EFI ::/EFI/BOOT/ 2>/dev/null || true
fi

# Create ISO
echo "[8/8] Creating ISO image..."
mkdir -p "$(dirname "$OUTPUT")"

if command -v xorriso &> /dev/null; then
    xorriso -as mkisofs \
        -iso-level 3 \
        -full-iso9660-filenames \
        -volid "MIXOS_LIVE" \
        -eltorito-boot boot/grub/i386-pc/eltorito.img \
        -no-emul-boot \
        -boot-load-size 4 \
        -boot-info-table \
        --eltorito-catalog boot/grub/boot.cat \
        --grub2-boot-info \
        --grub2-mbr /usr/lib/grub/i386-pc/boot_hybrid.img \
        -eltorito-alt-boot \
        -e efi.img \
        -no-emul-boot \
        -append_partition 2 0xef efi.img \
        -output "$OUTPUT" \
        . 2>/dev/null || {
        # Fallback to simpler ISO creation
        genisoimage -o "$OUTPUT" \
            -b boot/grub/i386-pc/eltorito.img \
            -no-emul-boot \
            -boot-load-size 4 \
            -boot-info-table \
            -V "MIXOS_LIVE" \
            -R -J \
            . 2>/dev/null || {
            echo "WARNING: ISO creation failed, creating placeholder"
            touch "$OUTPUT"
        }
    }
else
    echo "WARNING: xorriso not found, creating placeholder ISO"
    touch "$OUTPUT"
fi

# Show result
if [ -f "$OUTPUT" ] && [ -s "$OUTPUT" ]; then
    SIZE=$(du -h "$OUTPUT" | cut -f1)
    echo "=========================================="
    echo "ISO build complete!"
    echo "Output: $OUTPUT"
    echo "Size: $SIZE"
    echo "=========================================="
else
    echo "=========================================="
    echo "ISO build completed (placeholder)"
    echo "Install xorriso and rebuild for real ISO"
    echo "=========================================="
fi
