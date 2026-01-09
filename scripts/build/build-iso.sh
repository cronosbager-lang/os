#!/bin/bash
# ============================================================================
# MIXOS GO - ISO Build Script
# ============================================================================
#
# This script assembles the final bootable ISO image.
# The ISO supports both BIOS (isolinux) and UEFI (GRUB) boot.
#
# ISO Structure:
#   /boot/
#     vmlinuz-X.X.X-mixos       - Kernel image
#     initramfs-X.X.X-mixos.img - Initramfs with AI
#   /mixos/
#     rootfs.sfs                - Squashfs root filesystem
#   /EFI/BOOT/
#     BOOTx64.EFI               - UEFI bootloader
#     grub.cfg                  - GRUB config
#   /isolinux/
#     isolinux.bin              - BIOS bootloader
#     isolinux.cfg              - Isolinux config
#
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source config if not already set
if [ -z "${VERSION:-}" ]; then
    VERSION="${VERSION:-1.0.0}"
    KERNEL_VERSION="${KERNEL_VERSION:-6.6.10}"
    KERNEL_LOCALVERSION="${KERNEL_LOCALVERSION:--mixos}"
    KERNEL_FULL_VERSION="${KERNEL_VERSION}${KERNEL_LOCALVERSION}"
    ARCH="${ARCH:-x86_64}"
    BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"
    KERNEL_BUILD_DIR="${KERNEL_BUILD_DIR:-$BUILD_DIR/kernel}"
    INITRAMFS_BUILD_DIR="${INITRAMFS_BUILD_DIR:-$BUILD_DIR/initramfs}"
    ROOTFS_BUILD_DIR="${ROOTFS_BUILD_DIR:-$BUILD_DIR/rootfs}"
    ISO_BUILD_DIR="${ISO_BUILD_DIR:-$BUILD_DIR/iso}"
    OUTPUT_DIR="${OUTPUT_DIR:-$BUILD_DIR/output}"
fi

# Source directories
ISO_SRC_DIR="${ROOT_DIR}/iso"

# Input files
KERNEL_IMAGE="${KERNEL_BUILD_DIR}/vmlinuz-${KERNEL_FULL_VERSION}"
INITRAMFS_IMAGE="${INITRAMFS_BUILD_DIR}/initramfs-${KERNEL_FULL_VERSION}.img"
ROOTFS_SQUASHFS="${ROOTFS_BUILD_DIR}/rootfs.sfs"

# Output
ISO_WORK="${ISO_BUILD_DIR}/work"
ISO_IMAGE="${OUTPUT_DIR}/mixos-${VERSION}-${ARCH}.iso"

# Labels
LIVE_LABEL="${LIVE_LABEL:-MIXOS_LIVE}"
VOLUME_ID="MIXOS_${VERSION}"

# ============================================================================
# COLORS
# ============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()    { echo -e "${CYAN}[STEP]${NC} $1"; }

# ============================================================================
# FUNCTIONS
# ============================================================================

check_prerequisites() {
    log_step "Checking prerequisites..."
    
    local errors=0
    
    # Check input files
    if [ ! -f "$KERNEL_IMAGE" ]; then
        log_error "Kernel image not found: $KERNEL_IMAGE"
        errors=$((errors + 1))
    fi
    
    if [ ! -f "$INITRAMFS_IMAGE" ]; then
        log_error "Initramfs not found: $INITRAMFS_IMAGE"
        errors=$((errors + 1))
    fi
    
    if [ ! -f "$ROOTFS_SQUASHFS" ]; then
        log_error "Rootfs squashfs not found: $ROOTFS_SQUASHFS"
        errors=$((errors + 1))
    fi
    
    # Check tools
    if ! command -v xorriso &>/dev/null; then
        log_error "xorriso not found (install xorriso package)"
        errors=$((errors + 1))
    fi
    
    if [ $errors -gt 0 ]; then
        log_error "Prerequisites check failed"
        exit 1
    fi
    
    log_success "Prerequisites OK"
}

create_iso_structure() {
    log_step "[1/6] Creating ISO structure..."
    
    # Clean and create work directory
    rm -rf "$ISO_WORK"
    mkdir -p "$ISO_WORK"
    
    cd "$ISO_WORK"
    
    # Create directory structure
    mkdir -p boot
    mkdir -p mixos
    mkdir -p EFI/BOOT
    mkdir -p isolinux
    
    log_success "ISO structure created"
}

install_kernel_and_initramfs() {
    log_step "[2/6] Installing kernel and initramfs..."
    
    cd "$ISO_WORK"
    
    # Copy kernel
    cp "$KERNEL_IMAGE" boot/vmlinuz
    log_info "Installed: boot/vmlinuz"
    
    # Copy initramfs
    cp "$INITRAMFS_IMAGE" boot/initramfs.img
    log_info "Installed: boot/initramfs.img"
    
    log_success "Kernel and initramfs installed"
}

install_rootfs() {
    log_step "[3/6] Installing root filesystem..."
    
    cd "$ISO_WORK"
    
    # Copy squashfs
    cp "$ROOTFS_SQUASHFS" mixos/rootfs.sfs
    
    local size=$(du -h mixos/rootfs.sfs | cut -f1)
    log_info "Installed: mixos/rootfs.sfs ($size)"
    
    log_success "Root filesystem installed"
}

setup_isolinux() {
    log_step "[4/6] Setting up ISOLINUX (BIOS boot)..."
    
    cd "$ISO_WORK"
    
    # Copy isolinux files from source
    local isolinux_src="${ISO_SRC_DIR}/isolinux"
    
    # Try to find isolinux.bin from system
    local isolinux_bin=""
    for path in /usr/lib/ISOLINUX/isolinux.bin \
                /usr/share/syslinux/isolinux.bin \
                /usr/lib/syslinux/bios/isolinux.bin; do
        if [ -f "$path" ]; then
            isolinux_bin="$path"
            break
        fi
    done
    
    if [ -n "$isolinux_bin" ]; then
        cp "$isolinux_bin" isolinux/
        log_info "Installed: isolinux/isolinux.bin"
        
        # Copy additional syslinux modules
        local syslinux_dir=$(dirname "$isolinux_bin")
        for mod in ldlinux.c32 libutil.c32 libcom32.c32 vesamenu.c32 \
                   menu.c32 hdt.c32 reboot.c32 poweroff.c32; do
            if [ -f "$syslinux_dir/$mod" ]; then
                cp "$syslinux_dir/$mod" isolinux/
            fi
        done
    else
        log_warn "isolinux.bin not found, BIOS boot may not work"
    fi
    
    # Create isolinux.cfg
    cat > isolinux/isolinux.cfg << EOF
# MIXOS GO ISOLINUX Configuration
# BIOS boot menu

DEFAULT mixos
TIMEOUT 50
PROMPT 1

UI vesamenu.c32
MENU TITLE MIXOS GO ${VERSION} Boot Menu
MENU BACKGROUND splash.png
MENU COLOR border       30;44   #40ffffff #a0000000 std
MENU COLOR title        1;36;44 #9033ccff #a0000000 std
MENU COLOR sel          7;37;40 #e0ffffff #20ffffff all
MENU COLOR unsel        37;44   #50ffffff #a0000000 std
MENU COLOR help         37;40   #c0ffffff #a0000000 std
MENU COLOR timeout_msg  37;40   #80ffffff #00000000 std
MENU COLOR timeout      1;37;40 #c0ffffff #00000000 std

LABEL mixos
    MENU LABEL ^MIXOS GO
    MENU DEFAULT
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=live:LABEL=${LIVE_LABEL} rd.live.image quiet splash
    TEXT HELP
    Boot MIXOS GO operating system
    ENDTEXT

LABEL mixos-safe
    MENU LABEL MIXOS GO (^Safe Mode)
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=live:LABEL=${LIVE_LABEL} rd.live.image single
    TEXT HELP
    Boot MIXOS GO in safe mode (single user)
    ENDTEXT

LABEL mixos-debug
    MENU LABEL MIXOS GO (^Debug)
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=live:LABEL=${LIVE_LABEL} rd.live.image mixos.debug=1
    TEXT HELP
    Boot MIXOS GO with debug output
    ENDTEXT

LABEL mixos-installer
    MENU LABEL MIXOS GO ^Installer
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=live:LABEL=${LIVE_LABEL} rd.live.image mixos.installer=1 quiet
    TEXT HELP
    Start MIXOS GO installation wizard
    ENDTEXT

LABEL mixos-toram
    MENU LABEL MIXOS GO (Load to ^RAM)
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=live:LABEL=${LIVE_LABEL} rd.live.image rd.live.ram=1 quiet splash
    TEXT HELP
    Load entire system to RAM (requires 4GB+ RAM)
    ENDTEXT

LABEL reboot
    MENU LABEL ^Reboot
    COM32 reboot.c32

LABEL poweroff
    MENU LABEL ^Power Off
    COM32 poweroff.c32
EOF

    log_success "ISOLINUX configured"
}

setup_uefi() {
    log_step "[5/6] Setting up UEFI boot..."
    
    cd "$ISO_WORK"
    
    # Create GRUB config for UEFI
    cat > EFI/BOOT/grub.cfg << EOF
# MIXOS GO GRUB Configuration (UEFI)

set default=0
set timeout=5
set gfxmode=auto
set gfxpayload=keep

insmod all_video
insmod gfxterm
insmod png

terminal_output gfxterm

menuentry "MIXOS GO" --class mixos --class os {
    echo "Loading MIXOS GO..."
    linux /boot/vmlinuz root=live:LABEL=${LIVE_LABEL} rd.live.image quiet splash
    initrd /boot/initramfs.img
}

menuentry "MIXOS GO (Safe Mode)" --class mixos --class os {
    echo "Loading MIXOS GO in safe mode..."
    linux /boot/vmlinuz root=live:LABEL=${LIVE_LABEL} rd.live.image single
    initrd /boot/initramfs.img
}

menuentry "MIXOS GO (Debug)" --class mixos --class os {
    echo "Loading MIXOS GO with debug output..."
    linux /boot/vmlinuz root=live:LABEL=${LIVE_LABEL} rd.live.image mixos.debug=1
    initrd /boot/initramfs.img
}

menuentry "MIXOS GO Installer" --class mixos --class os {
    echo "Starting MIXOS GO Installer..."
    linux /boot/vmlinuz root=live:LABEL=${LIVE_LABEL} rd.live.image mixos.installer=1 quiet
    initrd /boot/initramfs.img
}

menuentry "MIXOS GO (Load to RAM)" --class mixos --class os {
    echo "Loading MIXOS GO to RAM..."
    linux /boot/vmlinuz root=live:LABEL=${LIVE_LABEL} rd.live.image rd.live.ram=1 quiet splash
    initrd /boot/initramfs.img
}

menuentry "Reboot" --class reboot {
    reboot
}

menuentry "Shutdown" --class shutdown {
    halt
}
EOF

    # Create EFI boot image
    local efi_img="$ISO_WORK/EFI/BOOT/efiboot.img"
    
    # Try to find or create BOOTX64.EFI
    local grub_efi=""
    for path in /usr/lib/grub/x86_64-efi/monolithic/grubx64.efi \
                /usr/share/grub/x86_64-efi/monolithic/grubx64.efi \
                /boot/efi/EFI/*/grubx64.efi; do
        if [ -f "$path" ]; then
            grub_efi="$path"
            break
        fi
    done
    
    if [ -n "$grub_efi" ]; then
        cp "$grub_efi" EFI/BOOT/BOOTX64.EFI
        log_info "Installed: EFI/BOOT/BOOTX64.EFI"
    else
        # Try to create GRUB EFI image
        if command -v grub-mkimage &>/dev/null; then
            log_info "Creating GRUB EFI image..."
            grub-mkimage -o EFI/BOOT/BOOTX64.EFI \
                -p /EFI/BOOT \
                -O x86_64-efi \
                fat iso9660 part_gpt part_msdos normal boot linux \
                configfile loopback chain efifwsetup efi_gop efi_uga \
                ls search search_label search_fs_uuid search_fs_file \
                gfxterm gfxterm_background gfxterm_menu test all_video \
                loadenv exfat ext2 2>/dev/null || log_warn "Failed to create GRUB EFI image"
        else
            log_warn "grub-mkimage not found, UEFI boot may not work"
        fi
    fi
    
    # Create EFI boot image (FAT filesystem)
    if command -v mformat &>/dev/null && [ -f "EFI/BOOT/BOOTX64.EFI" ]; then
        log_info "Creating EFI boot image..."
        
        # Calculate size needed
        local efi_size=$(du -sk EFI | cut -f1)
        efi_size=$((efi_size + 1024))  # Add 1MB padding
        
        # Create FAT image
        dd if=/dev/zero of="$efi_img" bs=1K count=$efi_size 2>/dev/null
        mformat -i "$efi_img" -F ::
        mcopy -i "$efi_img" -s EFI ::
        
        log_info "Created: EFI/BOOT/efiboot.img"
    fi
    
    log_success "UEFI boot configured"
}

create_iso_image() {
    log_step "[6/6] Creating ISO image..."
    
    mkdir -p "$OUTPUT_DIR"
    
    cd "$ISO_WORK"
    
    # Build xorriso command
    local xorriso_opts=(
        -as mkisofs
        -iso-level 3
        -full-iso9660-filenames
        -volid "$LIVE_LABEL"
        -publisher "MIXOS Project"
        -preparer "MIXOS Build System"
        -appid "MIXOS GO ${VERSION}"
    )
    
    # Add BIOS boot if isolinux available
    if [ -f "isolinux/isolinux.bin" ]; then
        xorriso_opts+=(
            -b isolinux/isolinux.bin
            -c isolinux/boot.cat
            -no-emul-boot
            -boot-load-size 4
            -boot-info-table
            -isohybrid-mbr /usr/lib/ISOLINUX/isohdpfx.bin 2>/dev/null || true
        )
    fi
    
    # Add UEFI boot if available
    if [ -f "EFI/BOOT/efiboot.img" ]; then
        xorriso_opts+=(
            -eltorito-alt-boot
            -e EFI/BOOT/efiboot.img
            -no-emul-boot
            -isohybrid-gpt-basdat
        )
    elif [ -f "EFI/BOOT/BOOTX64.EFI" ]; then
        xorriso_opts+=(
            -eltorito-alt-boot
            -e EFI/BOOT/BOOTX64.EFI
            -no-emul-boot
        )
    fi
    
    # Output
    xorriso_opts+=(
        -output "$ISO_IMAGE"
        .
    )
    
    log_info "Running xorriso..."
    xorriso "${xorriso_opts[@]}" 2>&1 | grep -v "^xorriso" || true
    
    if [ -f "$ISO_IMAGE" ]; then
        log_success "ISO image created"
    else
        log_error "Failed to create ISO image"
        exit 1
    fi
}

show_summary() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " ISO Build Complete"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Version:     ${VERSION}"
    echo " Kernel:      ${KERNEL_FULL_VERSION}"
    echo " Output:      ${ISO_IMAGE}"
    echo ""
    
    if [ -f "$ISO_IMAGE" ]; then
        local iso_size=$(du -h "$ISO_IMAGE" | cut -f1)
        local iso_bytes=$(stat -c%s "$ISO_IMAGE")
        
        echo " ISO Size:    $iso_size ($iso_bytes bytes)"
        echo " Volume ID:   $LIVE_LABEL"
        echo ""
        echo " Boot Support:"
        [ -f "$ISO_WORK/isolinux/isolinux.bin" ] && echo "   ✓ BIOS (ISOLINUX)"
        [ -f "$ISO_WORK/EFI/BOOT/BOOTX64.EFI" ] && echo "   ✓ UEFI (GRUB)"
        echo ""
        echo " Contents:"
        echo "   - Kernel: boot/vmlinuz"
        echo "   - Initramfs: boot/initramfs.img"
        echo "   - Rootfs: mixos/rootfs.sfs"
    fi
    
    echo ""
    echo " To test with QEMU:"
    echo "   qemu-system-x86_64 -cdrom $ISO_IMAGE -m 4G -enable-kvm"
    echo ""
    echo " To write to USB:"
    echo "   sudo dd if=$ISO_IMAGE of=/dev/sdX bs=4M status=progress"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " MIXOS GO - ISO Build"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Version:     ${VERSION}"
    echo " Kernel:      ${KERNEL_FULL_VERSION}"
    echo " Output:      ${ISO_IMAGE}"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    
    check_prerequisites
    create_iso_structure
    install_kernel_and_initramfs
    install_rootfs
    setup_isolinux
    setup_uefi
    create_iso_image
    show_summary
}

# Run main
main "$@"
