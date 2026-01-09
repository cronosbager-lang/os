#!/bin/bash
# ============================================================================
# MIXOS GO - Initramfs Build Script
# ============================================================================
#
# This script builds the initramfs with AI components for early boot.
#
# Contents of initramfs:
#   /init                    - Main init script
#   /bin/busybox             - All-in-one Unix tools
#   /bin/mix-agent-early     - AI agent for early boot
#   /lib/modules/            - Essential kernel modules only
#   /ai/model/               - Quantized AI model
#   /ai/prompts/             - AI prompts
#   /ai/tools/               - AI helper tools
#   /etc/                    - Configuration files
#
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Set defaults for build variables (avoid unbound variables with set -u)
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10}"
KERNEL_LOCALVERSION="${KERNEL_LOCALVERSION:--mixos}"
# Allow an externally-provided KERNEL_FULL_VERSION, otherwise build it from
# version + localversion.
KERNEL_FULL_VERSION="${KERNEL_FULL_VERSION:-${KERNEL_VERSION}${KERNEL_LOCALVERSION}}"

# Build directories (use existing values if already exported)
BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"
KERNEL_BUILD_DIR="${KERNEL_BUILD_DIR:-$BUILD_DIR/kernel}"
INITRAMFS_BUILD_DIR="${INITRAMFS_BUILD_DIR:-$BUILD_DIR/initramfs}"
PACKAGES_BUILD_DIR="${PACKAGES_BUILD_DIR:-$BUILD_DIR/packages}"
CACHE_DIR="${CACHE_DIR:-$BUILD_DIR/cache}"

# Source directories
INITRAMFS_SRC_DIR="${ROOT_DIR}/initramfs"

# Work directory
WORK_DIR="${INITRAMFS_BUILD_DIR}/work"

# Output
INITRAMFS_IMAGE="${INITRAMFS_BUILD_DIR}/initramfs-${KERNEL_FULL_VERSION}.img"

# Kernel modules source
KERNEL_MODULES_SRC="${KERNEL_BUILD_DIR}/modules/lib/modules/${KERNEL_FULL_VERSION}"

# Package binaries
MIX_AGENT_EARLY_BIN="${PACKAGES_BUILD_DIR}/mix-agent-early/mix-agent-early"

# AI model (would be downloaded separately in real build)
AI_MODEL_EARLY="${CACHE_DIR}/models/mix-early-q4_k_m.gguf"

# Essential modules for initramfs
ESSENTIAL_MODULES=(
    # Core kernel infrastructure
    crc16
    crc32c
    # Block layer and SCSI
    scsi_mod
    sd_mod
    sr_mod
    cdrom
    # SATA/AHCI
    libata
    ahci
    # NVMe
    nvme_core
    nvme
    # Filesystems
    jbd2
    ext4
    squashfs
    overlay
    loop
    isofs
    vfat
    fat
    nls_cp437
    nls_ascii
    nls_utf8
    # USB
    usbcore
    usb_common
    usb_storage
    uas
    xhci_hcd
    xhci_pci
    ehci_hcd
    ehci_pci
    ohci_hcd
    ohci_pci
    uhci_hcd
    # Virtio (for QEMU testing)
    virtio_blk
    virtio_pci
    virtio_scsi
    virtio_net
    # Network
    mii
    e1000
    e1000e
    r8169
    # MDI/MDIO for network
    libphy
    mdio_bus
    # Cache and memory
    mbcache
)

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
    
    # Check kernel modules
    if [ ! -d "$KERNEL_MODULES_SRC" ]; then
        log_error "Kernel modules not found: $KERNEL_MODULES_SRC"
        log_error "Run 'make kernel' first"
        errors=$((errors + 1))
    fi
    
    # Check mix-agent-early
    if [ ! -f "$MIX_AGENT_EARLY_BIN" ]; then
        log_warn "mix-agent-early not found: $MIX_AGENT_EARLY_BIN"
        log_warn "AI agent will not be included in initramfs"
    fi
    
    # Check busybox
    if ! command -v busybox &>/dev/null; then
        log_error "busybox not found in PATH"
        errors=$((errors + 1))
    fi
    
    # Check cpio
    if ! command -v cpio &>/dev/null; then
        log_error "cpio not found in PATH"
        errors=$((errors + 1))
    fi
    
    if [ $errors -gt 0 ]; then
        log_error "Prerequisites check failed"
        exit 1
    fi
    
    log_success "Prerequisites OK"
}

create_directory_structure() {
    log_step "[1/8] Creating directory structure..."
    
    # Clean and create work directory
    rm -rf "$WORK_DIR"
    mkdir -p "$WORK_DIR"
    
    cd "$WORK_DIR"
    
    # Create FHS-like structure
    mkdir -p bin sbin usr/bin usr/sbin
    mkdir -p lib lib64 usr/lib usr/lib64
    mkdir -p etc etc/mixos
    mkdir -p dev proc sys run tmp
    mkdir -p newroot
    mkdir -p var/log var/run
    
    # Create AI directories
    mkdir -p ai/model ai/prompts ai/tools
    
    # Create MIXOS directories
    mkdir -p opt/mixos/bin
    
    log_success "Directory structure created"
}

install_busybox() {
    log_step "[2/8] Installing busybox..."
    
    cd "$WORK_DIR"
    
    # Copy busybox
    local busybox_path=$(which busybox)
    cp "$busybox_path" bin/busybox
    chmod 755 bin/busybox
    
    # Create symlinks for essential applets
    local applets=(
        # Shell
        sh ash
        # File operations
        cat cp mv rm mkdir rmdir ln ls chmod chown touch
        # Text processing
        grep sed awk cut head tail sort uniq wc tr
        # System
        mount umount switch_root pivot_root
        modprobe insmod lsmod rmmod
        mknod
        # Utilities
        echo printf sleep
        dmesg
        reboot poweroff halt
        # Block devices
        blkid findfs
        losetup
        # Filesystem
        fsck
        # Network (basic)
        ip ifconfig hostname
        # Archive
        cpio gzip gunzip
        # Misc
        true false test '[' '[['
        env
        kill
        ps
        free
        df
        du
        uname
        date
        clear
    )
    
    for applet in "${applets[@]}"; do
        ln -sf busybox "bin/$applet"
    done
    
    # Also create in sbin for compatibility
    for applet in mount umount switch_root modprobe insmod blkid; do
        ln -sf ../bin/busybox "sbin/$applet"
    done
    
    log_success "Busybox installed with $(echo ${#applets[@]}) applets"
}

install_libc() {
    log_step "[2.5/8] Installing libc and runtime libraries..."
    
    cd "$WORK_DIR"
    
    # Find and copy libc
    # It may be in /lib64, /lib, or /lib/x86_64-linux-gnu
    local libc_paths=(
        "/lib/x86_64-linux-gnu/libc.so.6"
        "/lib64/libc.so.6"
        "/lib/libc.so.6"
    )
    
    for libc_path in "${libc_paths[@]}"; do
        if [ -f "$libc_path" ]; then
            cp "$libc_path" lib64/
            log_info "Copied libc: $(basename $libc_path)"
            break
        fi
    done
    
    # Find and copy ld-linux
    local ld_paths=(
        "/lib64/ld-linux-x86-64.so.2"
        "/lib/ld-linux-x86-64.so.2"
        "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2"
    )
    
    for ld_path in "${ld_paths[@]}"; do
        if [ -f "$ld_path" ]; then
            cp "$ld_path" lib64/
            log_info "Copied ld-linux: $(basename $ld_path)"
            break
        fi
    done
    
    # Copy other essential libc components if available
    for libname in libm.so libdl.so libnsl.so libpthread.so; do
        for libpath in /lib/x86_64-linux-gnu/$libname.6 /lib64/$libname.6 /lib/$libname.6; do
            if [ -f "$libpath" ]; then
                cp "$libpath" lib64/ 2>/dev/null || true
                log_info "Copied: $(basename $libpath)"
            fi
        done
    done

    # Ensure resolver and NSS libraries are included (libresolv, libnss_*)
    for lib in libresolv.so.2 libnss_files.so.2 libnss_dns.so.2 libnss_mdns4_minimal.so.2; do
        for p in /lib/x86_64-linux-gnu/$lib /lib64/$lib /lib/$lib; do
            if [ -f "$p" ]; then
                cp "$p" lib64/ 2>/dev/null || true
                log_info "Copied resolver/NSS lib: $(basename $p)"
                break
            fi
        done
    done
    
    # Also expose important libs under /lib for compatibility
    mkdir -p lib
    for f in ld-linux-x86-64.so.2 libc.so.6 libresolv.so.2 libm.so.6 libnss_files.so.2 libnss_dns.so.2; do
        if [ -f "lib64/$f" ]; then
            ln -sf /lib64/$f lib/$f || true
            log_info "Linked /lib/$f -> /lib64/$f"
        fi
    done

    log_success "Runtime libraries installed"
}

install_kernel_modules() {
    log_step "[3/8] Installing kernel modules..."
    
    cd "$WORK_DIR"
    
    if [ ! -d "$KERNEL_MODULES_SRC" ]; then
        log_warn "Kernel modules not found, skipping"
        return
    fi
    
    # Create modules directory
    mkdir -p "lib/modules/${KERNEL_FULL_VERSION}"
    
    local modules_installed=0
    local modules_missing=0
    
    # Copy essential modules
    for mod in "${ESSENTIAL_MODULES[@]}"; do
        # Find module file (could be .ko or .ko.xz or .ko.zst)
        local mod_file=$(find "$KERNEL_MODULES_SRC" -name "${mod}.ko*" -type f 2>/dev/null | head -1)
        
        if [ -n "$mod_file" ] && [ -f "$mod_file" ]; then
            # Get relative path within modules directory
            local rel_path="${mod_file#$KERNEL_MODULES_SRC/}"
            local dest_dir="lib/modules/${KERNEL_FULL_VERSION}/$(dirname "$rel_path")"
            
            mkdir -p "$dest_dir"
            cp "$mod_file" "$dest_dir/"
            modules_installed=$((modules_installed + 1))
        else
            log_warn "Module not found: $mod"
            modules_missing=$((modules_missing + 1))
        fi
    done
    
    # Copy modules.* files
    for f in modules.dep modules.dep.bin modules.alias modules.alias.bin \
             modules.symbols modules.symbols.bin modules.builtin \
             modules.builtin.bin modules.order modules.builtin.modinfo; do
        if [ -f "$KERNEL_MODULES_SRC/$f" ]; then
            cp "$KERNEL_MODULES_SRC/$f" "lib/modules/${KERNEL_FULL_VERSION}/"
        fi
    done
    
    # Regenerate modules.dep for our subset
    if command -v depmod &>/dev/null; then
        depmod -b "$WORK_DIR" "$KERNEL_FULL_VERSION" 2>/dev/null || true
    fi
    
    log_success "Installed $modules_installed modules ($modules_missing missing)"
}

install_mix_agent_early() {
    log_step "[4/8] Installing mix-agent-early..."
    
    cd "$WORK_DIR"
    
    if [ -f "$MIX_AGENT_EARLY_BIN" ]; then
        cp "$MIX_AGENT_EARLY_BIN" bin/mix-agent-early
        chmod 755 bin/mix-agent-early
        
        # Also link to /usr/bin for compatibility
        ln -sf ../../bin/mix-agent-early usr/bin/mix-agent-early
        
        log_success "mix-agent-early installed"
    else
        log_warn "mix-agent-early not found, creating placeholder"
        
        # Create a placeholder script
        cat > bin/mix-agent-early << 'PLACEHOLDER'
#!/bin/sh
echo "[mix-agent-early] AI agent not available"
echo "[mix-agent-early] Running in fallback mode"
exit 0
PLACEHOLDER
        chmod 755 bin/mix-agent-early
    fi
}

install_ai_components() {
    log_step "[5/8] Installing AI components..."
    
    cd "$WORK_DIR"
    
    # Install AI model (if available)
    if [ -f "$AI_MODEL_EARLY" ]; then
        cp "$AI_MODEL_EARLY" ai/model/
        log_info "AI model installed: $(basename "$AI_MODEL_EARLY")"
    else
        log_warn "AI model not found: $AI_MODEL_EARLY"
        log_warn "AI inference will not be available in early boot"
        
        # Create placeholder
        echo "# AI model placeholder" > ai/model/README
    fi
    
    # Install AI prompts
    local prompts_src="${INITRAMFS_SRC_DIR}/ai/prompts"
    if [ -d "$prompts_src" ]; then
        cp -r "$prompts_src"/* ai/prompts/ 2>/dev/null || true
    fi
    
    # Create default early boot prompt if not exists
    if [ ! -f "ai/prompts/early-boot.txt" ]; then
        cat > ai/prompts/early-boot.txt << 'PROMPT'
You are the MIXOS early boot AI agent. Your role is to:
1. Detect hardware and load appropriate kernel modules
2. Find and mount the root filesystem
3. Assist with boot troubleshooting

Available tools:
- detect_hardware: Scan system hardware
- load_module: Load a kernel module
- find_root: Search for root filesystem
- mount_fs: Mount a filesystem

Be concise and efficient. This is early boot - resources are limited.
PROMPT
    fi
    
    # Install AI tools
    local tools_src="${INITRAMFS_SRC_DIR}/ai/tools"
    if [ -d "$tools_src" ]; then
        for tool in "$tools_src"/*.sh; do
            if [ -f "$tool" ]; then
                cp "$tool" ai/tools/
                chmod 755 "ai/tools/$(basename "$tool")"
            fi
        done
    fi
    
    # Create essential AI tools if not present
    create_ai_tools
    
    log_success "AI components installed"
}

create_ai_tools() {
    # detect-hardware.sh
    if [ ! -f "ai/tools/detect-hardware.sh" ]; then
        cat > ai/tools/detect-hardware.sh << 'TOOL'
#!/bin/sh
# Hardware detection tool for AI agent

echo "{"
echo "  \"cpu\": {"
echo "    \"model\": \"$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | sed 's/^ //' || echo 'unknown')\","
echo "    \"cores\": $(grep -c '^processor' /proc/cpuinfo 2>/dev/null || echo 1)"
echo "  },"
echo "  \"memory\": {"
echo "    \"total_kb\": $(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo 0)"
echo "  },"
echo "  \"block_devices\": ["

first=1
for dev in /sys/block/*; do
    [ -d "$dev" ] || continue
    name=$(basename "$dev")
    case "$name" in loop*|ram*|dm-*) continue ;; esac
    
    [ $first -eq 0 ] && echo ","
    first=0
    
    size=$(cat "$dev/size" 2>/dev/null || echo 0)
    size_mb=$((size * 512 / 1024 / 1024))
    
    echo -n "    {\"name\": \"$name\", \"size_mb\": $size_mb}"
done

echo ""
echo "  ],"
echo "  \"boot_mode\": \"$([ -d /sys/firmware/efi ] && echo 'uefi' || echo 'bios')\""
echo "}"
TOOL
        chmod 755 ai/tools/detect-hardware.sh
    fi
    
    # load-module.sh
    if [ ! -f "ai/tools/load-module.sh" ]; then
        cat > ai/tools/load-module.sh << 'TOOL'
#!/bin/sh
# Module loading tool for AI agent

MODULE="$1"

if [ -z "$MODULE" ]; then
    echo '{"error": "No module specified"}'
    exit 1
fi

if modprobe "$MODULE" 2>/dev/null; then
    echo "{\"status\": \"loaded\", \"module\": \"$MODULE\"}"
else
    echo "{\"status\": \"failed\", \"module\": \"$MODULE\"}"
    exit 1
fi
TOOL
        chmod 755 ai/tools/load-module.sh
    fi
    
    # find-root.sh
    if [ ! -f "ai/tools/find-root.sh" ]; then
        cat > ai/tools/find-root.sh << 'TOOL'
#!/bin/sh
# Root filesystem finder for AI agent

echo "{"
echo "  \"candidates\": ["

first=1

# Check by label
for label in MIXOS_ROOT MIXOS_LIVE; do
    dev=$(blkid -L "$label" 2>/dev/null)
    if [ -n "$dev" ]; then
        [ $first -eq 0 ] && echo ","
        first=0
        echo -n "    {\"device\": \"$dev\", \"label\": \"$label\", \"priority\": 1}"
    fi
done

# Check common devices
for dev in /dev/sda2 /dev/vda2 /dev/nvme0n1p2 /dev/sr0; do
    if [ -b "$dev" ]; then
        [ $first -eq 0 ] && echo ","
        first=0
        fstype=$(blkid -o value -s TYPE "$dev" 2>/dev/null || echo "unknown")
        echo -n "    {\"device\": \"$dev\", \"fstype\": \"$fstype\", \"priority\": 2}"
    fi
done

echo ""
echo "  ]"
echo "}"
TOOL
        chmod 755 ai/tools/find-root.sh
    fi
}

install_config_files() {
    log_step "[6/8] Installing configuration files..."
    
    cd "$WORK_DIR"
    
    # modules.conf - modules to load at boot
    cat > etc/modules.conf << EOF
# Essential modules for MIXOS early boot
# These are loaded by the init script

# Storage
ahci
sd_mod
nvme

# Filesystems
ext4
squashfs
overlay
loop
isofs

# USB
usb_storage
xhci_hcd
ehci_hcd

# Virtio (for VM testing)
virtio_blk
virtio_pci
virtio_net
EOF

    # MIXOS early config
    cat > etc/mixos/early.conf << EOF
# MIXOS Early Boot Configuration

[boot]
# Enable AI-assisted boot
ai_enabled=true

# Timeout for root detection (seconds)
root_timeout=30

# Default root label
root_label=MIXOS_ROOT

# Live boot label
live_label=MIXOS_LIVE

[ai]
# AI model path
model_path=/ai/model/mix-early-q4_k_m.gguf

# Enable AI hardware detection
ai_detect_hardware=true

# Enable AI module loading
ai_load_modules=true

[debug]
# Enable debug output
debug=false

# Enable shell on failure
shell_on_fail=true
EOF

    # Create minimal passwd/group for switch_root
    cat > etc/passwd << 'EOF'
root:x:0:0:root:/root:/bin/sh
EOF

    cat > etc/group << 'EOF'
root:x:0:
EOF

    log_success "Configuration files installed"
}

install_init_script() {
    log_step "[7/8] Installing init script..."
    
    cd "$WORK_DIR"
    
    # Kernel cannot execute shell scripts directly - they need an interpreter
    # Solution: Create /init as hardlink to /bin/busybox, then it will act as sh
    # and we'll put the actual init logic in /init.sh which will be sourced
    
    if [ -f "${INITRAMFS_SRC_DIR}/init" ]; then
        # Copy actual init script as init.sh (to be sourced by /bin/sh)
        cp "${INITRAMFS_SRC_DIR}/init" init.sh
        chmod 755 init.sh
        log_info "Init script copied to init.sh (will be sourced by sh)"
    else
        # Create placeholder
        cat > init.sh << 'INIT'
#!/bin/sh
echo "MIXOS Initramfs - placeholder init"
exec /bin/sh
INIT
        chmod 755 init.sh
    fi
    
    # Create /init as symlink to /bin/sh
    # When kernel execs /init, it will actually be /bin/sh
    ln -sf /bin/sh init
    
    log_success "Init script setup complete (init -> /bin/sh, logic in init.sh)"

    # Try to create a small static init wrapper so kernel can exec an ELF
    # without relying on shared libraries. If static build fails, keep
    # the symlink to /bin/sh.
    cat > init.c << 'INIT_C'
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>

int main(int argc, char *argv[], char *envp[]) {
    pid_t pid = fork();
    if (pid == 0) {
        execve("/bin/sh", (char *const[]){"/bin/sh", "/init.sh", NULL}, envp);
        _exit(127);
    } else if (pid > 0) {
        int status;
        waitpid(pid, &status, 0);
        return WIFEXITED(status) ? WEXITSTATUS(status) : 1;
    } else {
        return 1;
    }
}
INIT_C

    if command -v gcc &>/dev/null; then
        if gcc -static -O2 -s -o init init.c >/dev/null 2>&1; then
            chmod 755 init || true
            log_info "Built static /init wrapper; replacing symlink"
        else
            log_warn "Static build of /init failed; keeping symlink to /bin/sh"
            rm -f init || true
            ln -sf /bin/sh init
        fi
    else
        log_warn "gcc not available; using /bin/sh as /init"
    fi

    # Clean up build artefact
    rm -f init.c || true
}

create_initramfs_image() {
    log_step "[8/8] Creating initramfs image..."
    
    cd "$WORK_DIR"
    
    # Create output directory
    mkdir -p "$(dirname "$INITRAMFS_IMAGE")"
    
    # Create cpio archive and compress
    log_info "Creating cpio archive..."
    find . -print0 | cpio --null --create --format=newc 2>/dev/null | \
        gzip -9 > "$INITRAMFS_IMAGE"
    
    log_success "Initramfs image created"
}

show_summary() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " Initramfs Build Complete"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Kernel Version: $KERNEL_FULL_VERSION"
    echo " Output:         $INITRAMFS_IMAGE"
    echo ""
    
    if [ -f "$INITRAMFS_IMAGE" ]; then
        local size=$(du -h "$INITRAMFS_IMAGE" | cut -f1)
        local size_bytes=$(stat -c%s "$INITRAMFS_IMAGE")
        echo " Size:           $size ($size_bytes bytes)"
    fi
    
    echo ""
    echo " Contents:"
    echo "   - Busybox with essential applets"
    
    if [ -f "$WORK_DIR/bin/mix-agent-early" ] && [ ! -h "$WORK_DIR/bin/mix-agent-early" ]; then
        echo "   - mix-agent-early (AI agent)"
    else
        echo "   - mix-agent-early (placeholder)"
    fi
    
    local mod_count=$(find "$WORK_DIR/lib/modules" -name "*.ko*" 2>/dev/null | wc -l)
    echo "   - $mod_count kernel modules"
    
    if [ -f "$WORK_DIR/ai/model/"*.gguf 2>/dev/null ]; then
        echo "   - AI model"
    else
        echo "   - AI model (not included)"
    fi
    
    echo "   - AI prompts and tools"
    echo "   - Configuration files"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " MIXOS GO - Initramfs Build"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Kernel Version:  $KERNEL_FULL_VERSION"
    echo " Work Directory:  $WORK_DIR"
    echo " Output:          $INITRAMFS_IMAGE"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    
    check_prerequisites
    create_directory_structure
    install_busybox
    install_libc
    install_kernel_modules
    install_mix_agent_early
    install_ai_components
    install_config_files
    install_init_script
    create_initramfs_image
    show_summary
}

# Run main
main "$@"
