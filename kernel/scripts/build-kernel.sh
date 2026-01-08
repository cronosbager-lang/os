#!/bin/bash
# MIXOS Kernel Build Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KERNEL_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$KERNEL_DIR")"
BUILD_DIR="$ROOT_DIR/build/kernel"

KERNEL_VERSION="${KERNEL_VERSION:-6.6.10}"
CONFIG="${CONFIG:-mixos-default}"

echo "=========================================="
echo "MIXOS Kernel Build"
echo "Version: $KERNEL_VERSION"
echo "Config: $CONFIG"
echo "=========================================="

cd "$KERNEL_DIR"

# Download
if [ ! -f "linux-$KERNEL_VERSION.tar.xz" ]; then
    echo "[1/5] Downloading kernel source..."
    MAJOR=$(echo $KERNEL_VERSION | cut -d. -f1)
    curl -L -o "linux-$KERNEL_VERSION.tar.xz" \
        "https://cdn.kernel.org/pub/linux/kernel/v${MAJOR}.x/linux-$KERNEL_VERSION.tar.xz"
else
    echo "[1/5] Kernel source already downloaded"
fi

# Extract
if [ ! -d "linux-$KERNEL_VERSION" ]; then
    echo "[2/5] Extracting kernel source..."
    tar xf "linux-$KERNEL_VERSION.tar.xz"
else
    echo "[2/5] Kernel source already extracted"
fi

# Configure
echo "[3/5] Configuring kernel..."
cd "linux-$KERNEL_VERSION"
cp "$KERNEL_DIR/config/$CONFIG.config" .config
make olddefconfig

# Apply patches if any
if [ -d "$KERNEL_DIR/patches" ] && [ "$(ls -A $KERNEL_DIR/patches/*.patch 2>/dev/null)" ]; then
    echo "[3.5/5] Applying patches..."
    for patch in "$KERNEL_DIR/patches"/*.patch; do
        echo "  Applying: $(basename $patch)"
        patch -p1 < "$patch" || true
    done
fi

# Build
echo "[4/5] Building kernel (this may take a while)..."
make -j$(nproc) bzImage
make -j$(nproc) modules

# Install
echo "[5/5] Installing to build directory..."
mkdir -p "$BUILD_DIR"
cp arch/x86/boot/bzImage "$BUILD_DIR/vmlinuz-$KERNEL_VERSION-mixos"

mkdir -p "$BUILD_DIR/modules"
make INSTALL_MOD_PATH="$BUILD_DIR/modules" modules_install

echo "=========================================="
echo "Kernel build complete!"
echo "Kernel: $BUILD_DIR/vmlinuz-$KERNEL_VERSION-mixos"
echo "Modules: $BUILD_DIR/modules/lib/modules/$KERNEL_VERSION-mixos/"
echo "=========================================="
