#!/bin/bash
# ============================================================================
# MIXOS GO - Dependency Checker
# ============================================================================
#
# Checks for all required build dependencies and reports status.
#
# ============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo "════════════════════════════════════════════════════════════════"
echo " MIXOS GO - Build Dependency Check"
echo "════════════════════════════════════════════════════════════════"
echo ""

errors=0
warnings=0

check_command() {
    local cmd="$1"
    local pkg="${2:-$1}"
    local required="${3:-yes}"
    
    if command -v "$cmd" &>/dev/null; then
        local version=$($cmd --version 2>/dev/null | head -1 || echo "unknown")
        printf "  ${GREEN}✓${NC} %-20s %s\n" "$cmd" "$version"
        return 0
    else
        if [ "$required" = "yes" ]; then
            printf "  ${RED}✗${NC} %-20s (install: %s)\n" "$cmd" "$pkg"
            errors=$((errors + 1))
        else
            printf "  ${YELLOW}○${NC} %-20s (optional: %s)\n" "$cmd" "$pkg"
            warnings=$((warnings + 1))
        fi
        return 1
    fi
}

echo "Essential Build Tools:"
echo "──────────────────────────────────────────────────────────────────"
check_command make
check_command gcc
check_command g++ "g++"
check_command ld "binutils"
check_command ar "binutils"
check_command git
check_command curl
check_command tar
check_command gzip
check_command xz "xz-utils"
check_command cpio

echo ""
echo "Kernel Build Dependencies:"
echo "──────────────────────────────────────────────────────────────────"
check_command bc
check_command flex
check_command bison
check_command perl

# Check for libelf
if pkg-config --exists libelf 2>/dev/null || [ -f /usr/include/libelf.h ]; then
    printf "  ${GREEN}✓${NC} %-20s\n" "libelf"
else
    printf "  ${RED}✗${NC} %-20s (install: libelf-dev)\n" "libelf"
    errors=$((errors + 1))
fi

# Check for libssl
if pkg-config --exists openssl 2>/dev/null || [ -f /usr/include/openssl/ssl.h ]; then
    printf "  ${GREEN}✓${NC} %-20s\n" "libssl"
else
    printf "  ${RED}✗${NC} %-20s (install: libssl-dev)\n" "libssl"
    errors=$((errors + 1))
fi

echo ""
echo "Go Build Dependencies:"
echo "──────────────────────────────────────────────────────────────────"
check_command go "golang-go"

echo ""
echo "Rust Build Dependencies:"
echo "──────────────────────────────────────────────────────────────────"
check_command cargo "rustup"
check_command rustc "rustup"

echo ""
echo "Python Dependencies:"
echo "──────────────────────────────────────────────────────────────────"
check_command python3
check_command pip3 "python3-pip"

echo ""
echo "Filesystem Tools:"
echo "──────────────────────────────────────────────────────────────────"
check_command mksquashfs "squashfs-tools"
check_command unsquashfs "squashfs-tools"
check_command mkfs.ext4 "e2fsprogs"
check_command mkfs.vfat "dosfstools"

echo ""
echo "ISO Build Tools:"
echo "──────────────────────────────────────────────────────────────────"
check_command xorriso
check_command grub-mkimage "grub-common" "no"
check_command mformat "mtools" "no"

echo ""
echo "Bootloader Tools:"
echo "──────────────────────────────────────────────────────────────────"
# Check for isolinux
if [ -f /usr/lib/ISOLINUX/isolinux.bin ] || [ -f /usr/share/syslinux/isolinux.bin ]; then
    printf "  ${GREEN}✓${NC} %-20s\n" "isolinux"
else
    printf "  ${YELLOW}○${NC} %-20s (install: isolinux syslinux-common)\n" "isolinux"
    warnings=$((warnings + 1))
fi

echo ""
echo "Testing Tools:"
echo "──────────────────────────────────────────────────────────────────"
check_command qemu-system-x86_64 "qemu-system-x86" "no"
check_command busybox "busybox" "no"

echo ""
echo "════════════════════════════════════════════════════════════════"

if [ $errors -gt 0 ]; then
    echo -e " ${RED}Status: $errors required dependencies missing${NC}"
    echo ""
    echo " Install missing dependencies:"
    echo "   Ubuntu/Debian:"
    echo "     sudo apt install build-essential bc flex bison libelf-dev libssl-dev"
    echo "     sudo apt install squashfs-tools xorriso isolinux grub-pc-bin"
    echo "     sudo apt install golang-go rustup python3 python3-pip"
    echo ""
    echo "   Arch Linux:"
    echo "     sudo pacman -S base-devel bc flex bison libelf openssl"
    echo "     sudo pacman -S squashfs-tools xorriso syslinux grub"
    echo "     sudo pacman -S go rust python python-pip"
    echo ""
    exit 1
elif [ $warnings -gt 0 ]; then
    echo -e " ${YELLOW}Status: All required dependencies found ($warnings optional missing)${NC}"
else
    echo -e " ${GREEN}Status: All dependencies found${NC}"
fi

echo "════════════════════════════════════════════════════════════════"
echo ""
