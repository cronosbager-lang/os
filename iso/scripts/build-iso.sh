#!/bin/bash
# MIXOS ISO Build Script
# Creates a bootable ISO image with BIOS and UEFI support

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISO_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$ISO_DIR")"
BUILD_DIR="$ROOT_DIR/build/iso"
WORK_DIR="$BUILD_DIR/work"

VERSION="${VERSION:-1.0}"
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10-mixos}"
ARCH="${ARCH:-x86_64}"
OUTPUT="$BUILD_DIR/mixos-go-$VERSION-$ARCH.iso"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "=========================================="
echo "MIXOS GO ISO Build"
echo "Version: $VERSION"
echo "Kernel: $KERNEL_VERSION"
echo "Architecture: $ARCH"
echo "=========================================="

# Clean and create work directory
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

# Create ISO directory structure
log_info "[1/10] Creating ISO structure..."
mkdir -p boot/grub/themes/mixos
mkdir -p boot/grub/i386-pc
mkdir -p boot/grub/x86_64-efi
mkdir -p EFI/BOOT
mkdir -p LiveOS
mkdir -p isolinux
mkdir -p mixos/{packages,ai-model}
log_success "Directory structure created"

# Copy kernel
log_info "[2/10] Copying kernel..."
KERNEL_PATH="$ROOT_DIR/build/kernel/vmlinuz-$KERNEL_VERSION"
if [ -f "$KERNEL_PATH" ]; then
    cp "$KERNEL_PATH" boot/vmlinuz-mixos
    log_success "Kernel copied"
else
    log_warn "Kernel not found at $KERNEL_PATH, creating placeholder"
    echo "MIXOS Kernel Placeholder" > boot/vmlinuz-mixos
fi

# Copy initramfs
log_info "[3/10] Copying initramfs..."
INITRAMFS_PATH="$ROOT_DIR/build/initramfs/initramfs-$KERNEL_VERSION.img"
if [ -f "$INITRAMFS_PATH" ]; then
    cp "$INITRAMFS_PATH" boot/initramfs-mixos.img
    log_success "Initramfs copied"
else
    log_warn "Initramfs not found at $INITRAMFS_PATH, creating placeholder"
    echo "MIXOS Initramfs Placeholder" > boot/initramfs-mixos.img
fi

# Create squashfs image from rootfs
log_info "[4/10] Creating squashfs image..."
ROOTFS_PATH="$ROOT_DIR/build/rootfs/work"
if [ -d "$ROOTFS_PATH" ] && [ "$(ls -A $ROOTFS_PATH 2>/dev/null)" ]; then
    if command -v mksquashfs &> /dev/null; then
        mksquashfs "$ROOTFS_PATH" LiveOS/squashfs.img \
            -comp zstd -Xcompression-level 19 \
            -b 1M -no-recovery \
            -processors $(nproc) \
            2>/dev/null && log_success "Squashfs image created" || {
            log_warn "mksquashfs failed, creating placeholder"
            dd if=/dev/zero of=LiveOS/squashfs.img bs=1M count=1 2>/dev/null
        }
    else
        log_warn "mksquashfs not found, creating placeholder"
        dd if=/dev/zero of=LiveOS/squashfs.img bs=1M count=1 2>/dev/null
    fi
else
    log_warn "RootFS not found at $ROOTFS_PATH, creating placeholder"
    dd if=/dev/zero of=LiveOS/squashfs.img bs=1M count=1 2>/dev/null
fi

# Copy GRUB configuration
log_info "[5/10] Setting up GRUB..."
cp "$ISO_DIR/grub/grub.cfg" boot/grub/
if [ -d "$ISO_DIR/themes/mixos" ]; then
    cp -r "$ISO_DIR/themes/mixos/"* boot/grub/themes/mixos/ 2>/dev/null || true
fi
log_success "GRUB configuration copied"

# Copy isolinux configuration
log_info "[6/10] Setting up ISOLINUX..."
if [ -f "$ISO_DIR/isolinux/isolinux.cfg" ]; then
    cp "$ISO_DIR/isolinux/isolinux.cfg" isolinux/
fi
# Copy isolinux binaries if available
for file in isolinux.bin ldlinux.c32 libutil.c32 libcom32.c32 vesamenu.c32 reboot.c32 poweroff.c32 hdt.c32; do
    if [ -f "/usr/lib/syslinux/bios/$file" ]; then
        cp "/usr/lib/syslinux/bios/$file" isolinux/
    elif [ -f "/usr/share/syslinux/$file" ]; then
        cp "/usr/share/syslinux/$file" isolinux/
    fi
done
log_success "ISOLINUX configuration set up"

# Create GRUB BIOS image
log_info "[7/10] Creating GRUB BIOS image..."
if command -v grub-mkimage &> /dev/null; then
    # Create core.img for BIOS
    grub-mkimage -o boot/grub/i386-pc/core.img \
        -O i386-pc \
        -p /boot/grub \
        biosdisk iso9660 part_gpt part_msdos fat ext2 \
        normal boot linux configfile loopback chain \
        ls search search_label search_fs_uuid search_fs_file \
        gfxterm gfxterm_background gfxterm_menu test all_video \
        loadenv 2>/dev/null && log_success "GRUB BIOS image created" || log_warn "GRUB BIOS image creation failed"
    
    # Create eltorito.img
    if [ -f /usr/lib/grub/i386-pc/cdboot.img ] && [ -f boot/grub/i386-pc/core.img ]; then
        cat /usr/lib/grub/i386-pc/cdboot.img boot/grub/i386-pc/core.img > boot/grub/i386-pc/eltorito.img
    fi
else
    log_warn "grub-mkimage not found"
fi

# Create GRUB EFI image
log_info "[8/10] Creating EFI boot image..."
if command -v grub-mkimage &> /dev/null; then
    grub-mkimage -o EFI/BOOT/BOOTX64.EFI \
        -O x86_64-efi \
        -p /boot/grub \
        part_gpt part_msdos fat ext2 iso9660 normal boot linux \
        configfile loopback chain efifwsetup efi_gop \
        efi_uga ls search search_label search_fs_uuid \
        search_fs_file gfxterm gfxterm_background \
        gfxterm_menu test all_video loadenv exfat \
        2>/dev/null && log_success "EFI boot image created" || log_warn "EFI boot image creation failed"
fi

# Copy EFI GRUB config
if [ -f "$ISO_DIR/efi/EFI/BOOT/grub.cfg" ]; then
    cp "$ISO_DIR/efi/EFI/BOOT/grub.cfg" EFI/BOOT/
fi

# Create EFI boot partition image
log_info "[9/10] Creating EFI partition image..."
dd if=/dev/zero of=efi.img bs=1M count=64 2>/dev/null
if command -v mkfs.vfat &> /dev/null; then
    mkfs.vfat -F 32 -n "MIXOS_EFI" efi.img 2>/dev/null || true
    if command -v mmd &> /dev/null && command -v mcopy &> /dev/null; then
        mmd -i efi.img ::/EFI ::/EFI/BOOT 2>/dev/null || true
        if [ -f EFI/BOOT/BOOTX64.EFI ]; then
            mcopy -i efi.img EFI/BOOT/BOOTX64.EFI ::/EFI/BOOT/ 2>/dev/null || true
        fi
        if [ -f EFI/BOOT/grub.cfg ]; then
            mcopy -i efi.img EFI/BOOT/grub.cfg ::/EFI/BOOT/ 2>/dev/null || true
        fi
        log_success "EFI partition image created"
    else
        log_warn "mtools not found, EFI partition may be incomplete"
    fi
else
    log_warn "mkfs.vfat not found"
fi

# Create ISO
log_info "[10/10] Creating ISO image..."
mkdir -p "$(dirname "$OUTPUT")"

if command -v xorriso &> /dev/null; then
    # Full hybrid ISO with BIOS and UEFI support
    XORRISO_ARGS=(
        -as mkisofs
        -iso-level 3
        -full-iso9660-filenames
        -joliet
        -joliet-long
        -rational-rock
        -volid "MIXOS_LIVE"
        -publisher "MIXOS Project"
        -preparer "MIXOS Build System"
    )
    
    # Add BIOS boot if eltorito.img exists
    if [ -f boot/grub/i386-pc/eltorito.img ]; then
        XORRISO_ARGS+=(
            -eltorito-boot boot/grub/i386-pc/eltorito.img
            -no-emul-boot
            -boot-load-size 4
            -boot-info-table
            --eltorito-catalog boot/grub/boot.cat
            --grub2-boot-info
        )
        # Add MBR if available
        if [ -f /usr/lib/grub/i386-pc/boot_hybrid.img ]; then
            XORRISO_ARGS+=(--grub2-mbr /usr/lib/grub/i386-pc/boot_hybrid.img)
        fi
    elif [ -f isolinux/isolinux.bin ]; then
        XORRISO_ARGS+=(
            -eltorito-boot isolinux/isolinux.bin
            -no-emul-boot
            -boot-load-size 4
            -boot-info-table
            --eltorito-catalog isolinux/boot.cat
        )
        if [ -f /usr/lib/syslinux/bios/isohdpfx.bin ]; then
            XORRISO_ARGS+=(-isohybrid-mbr /usr/lib/syslinux/bios/isohdpfx.bin)
        fi
    fi
    
    # Add EFI boot
    if [ -f efi.img ]; then
        XORRISO_ARGS+=(
            -eltorito-alt-boot
            -e efi.img
            -no-emul-boot
            -append_partition 2 0xef efi.img
        )
    fi
    
    XORRISO_ARGS+=(-output "$OUTPUT" .)
    
    xorriso "${XORRISO_ARGS[@]}" 2>/dev/null && log_success "ISO created with xorriso" || {
        log_warn "xorriso failed, trying genisoimage..."
        if command -v genisoimage &> /dev/null; then
            genisoimage -o "$OUTPUT" \
                -V "MIXOS_LIVE" \
                -R -J -l \
                -b isolinux/isolinux.bin \
                -c isolinux/boot.cat \
                -no-emul-boot \
                -boot-load-size 4 \
                -boot-info-table \
                . 2>/dev/null && log_success "ISO created with genisoimage" || {
                log_error "ISO creation failed"
                touch "$OUTPUT"
            }
        else
            log_error "No ISO creation tool available"
            touch "$OUTPUT"
        fi
    }
elif command -v genisoimage &> /dev/null; then
    genisoimage -o "$OUTPUT" \
        -V "MIXOS_LIVE" \
        -R -J -l \
        -b isolinux/isolinux.bin \
        -c isolinux/boot.cat \
        -no-emul-boot \
        -boot-load-size 4 \
        -boot-info-table \
        . 2>/dev/null && log_success "ISO created with genisoimage" || {
        log_error "ISO creation failed"
        touch "$OUTPUT"
    }
else
    log_error "No ISO creation tool found (xorriso or genisoimage required)"
    touch "$OUTPUT"
fi

# Make ISO hybrid (bootable from USB)
if [ -f "$OUTPUT" ] && [ -s "$OUTPUT" ]; then
    if command -v isohybrid &> /dev/null; then
        isohybrid --uefi "$OUTPUT" 2>/dev/null && log_success "ISO made hybrid (USB bootable)" || true
    fi
fi

# Show result
echo ""
echo "=========================================="
if [ -f "$OUTPUT" ] && [ -s "$OUTPUT" ]; then
    SIZE=$(du -h "$OUTPUT" | cut -f1)
    log_success "ISO build complete!"
    echo "Output: $OUTPUT"
    echo "Size: $SIZE"
    echo ""
    echo "To test with QEMU (BIOS):"
    echo "  qemu-system-x86_64 -cdrom $OUTPUT -m 2G -enable-kvm"
    echo ""
    echo "To test with QEMU (UEFI):"
    echo "  qemu-system-x86_64 -cdrom $OUTPUT -m 2G -enable-kvm -bios /usr/share/ovmf/OVMF.fd"
else
    log_warn "ISO build completed (placeholder)"
    echo "Install xorriso/genisoimage and rebuild for real ISO"
fi
echo "=========================================="
