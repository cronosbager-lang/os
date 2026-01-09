#!/bin/bash
# ============================================================================
# MIXOS GO - Root Filesystem Build Script
# ============================================================================
#
# This script builds the root filesystem for MIXOS GO.
# It creates a complete Linux system with:
#   - Base system (minimal Linux userspace)
#   - All kernel modules
#   - MIXOS packages (mix-cli, mix-pkg, mix-agent, etc.)
#   - Systemd services
#   - Configuration files
#   - AI model (full version)
#
# The output is a directory that can be compressed to squashfs.
#
# ============================================================================

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Set build configuration defaults (avoid unbound variables with set -u)
VERSION="${VERSION:-1.0.0}"
KERNEL_VERSION="${KERNEL_VERSION:-6.6.10}"
KERNEL_LOCALVERSION="${KERNEL_LOCALVERSION:--mixos}"
KERNEL_FULL_VERSION="${KERNEL_FULL_VERSION:-${KERNEL_VERSION}${KERNEL_LOCALVERSION}}"

BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build}"
KERNEL_BUILD_DIR="${KERNEL_BUILD_DIR:-$BUILD_DIR/kernel}"
ROOTFS_BUILD_DIR="${ROOTFS_BUILD_DIR:-$BUILD_DIR/rootfs}"
PACKAGES_BUILD_DIR="${PACKAGES_BUILD_DIR:-$BUILD_DIR/packages}"
CACHE_DIR="${CACHE_DIR:-$BUILD_DIR/cache}"

# Source directories
ROOTFS_SRC_DIR="${ROOT_DIR}/rootfs"
CONFIGS_SRC_DIR="${ROOT_DIR}/configs"

# Output directory
ROOTFS_ROOT="${ROOTFS_BUILD_DIR}/root"

# Kernel modules source
KERNEL_MODULES_SRC="${KERNEL_BUILD_DIR}/modules/lib/modules/${KERNEL_FULL_VERSION}"

# Package binaries
PKG_MIX_CLI="${PACKAGES_BUILD_DIR}/mix-cli/mix-cli"
PKG_MIX_PKG="${PACKAGES_BUILD_DIR}/mix-pkg/mix-pkg"
PKG_MIX_AGENT="${PACKAGES_BUILD_DIR}/mix-agent"
PKG_MIX_AGENT_EARLY="${PACKAGES_BUILD_DIR}/mix-agent-early/mix-agent-early"
PKG_MIX_INSTALLER="${PACKAGES_BUILD_DIR}/mix-installer/mix-installer"

# AI model
AI_MODEL_FULL="${CACHE_DIR}/models/mix-small-1.1b-q4_k_m.gguf"

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
    
    # Check packages
    local missing_pkgs=()
    [ ! -f "$PKG_MIX_CLI" ] && missing_pkgs+=("mix-cli")
    [ ! -f "$PKG_MIX_PKG" ] && missing_pkgs+=("mix-pkg")
    [ ! -d "$PKG_MIX_AGENT" ] && missing_pkgs+=("mix-agent")
    [ ! -f "$PKG_MIX_INSTALLER" ] && missing_pkgs+=("mix-installer")
    
    if [ ${#missing_pkgs[@]} -gt 0 ]; then
        log_warn "Missing packages: ${missing_pkgs[*]}"
        log_warn "Run 'make packages' first"
    fi
    
    if [ $errors -gt 0 ]; then
        log_error "Prerequisites check failed"
        exit 1
    fi
    
    log_success "Prerequisites OK"
}

create_directory_structure() {
    log_step "[1/10] Creating directory structure..."
    
    # Clean and create root directory
    rm -rf "$ROOTFS_ROOT"
    mkdir -p "$ROOTFS_ROOT"
    
    cd "$ROOTFS_ROOT"
    
    # Create FHS directory structure
    mkdir -p bin sbin lib lib64
    mkdir -p usr/bin usr/sbin usr/lib usr/lib64 usr/local/bin usr/local/sbin
    mkdir -p usr/share/man usr/share/doc usr/share/mixos
    mkdir -p etc/mixos etc/systemd/system etc/profile.d etc/skel
    mkdir -p var/log var/lib var/cache var/run var/tmp var/spool
    mkdir -p var/lib/mixos/packages var/lib/mixos/agent
    mkdir -p opt/mixos/ai/model opt/mixos/scripts opt/mixos/tools
    mkdir -p home root
    mkdir -p tmp
    mkdir -p proc sys dev run
    mkdir -p boot
    mkdir -p mnt media
    mkdir -p srv
    
    # Set permissions
    chmod 1777 tmp var/tmp
    chmod 700 root
    
    log_success "Directory structure created"
}

create_base_system() {
    log_step "[2/10] Creating base system files..."
    
    cd "$ROOTFS_ROOT"
    
    # /etc/os-release
    cat > etc/os-release << EOF
NAME="MIXOS GO"
VERSION="${VERSION}"
ID=mixos
ID_LIKE=arch
VERSION_ID=${VERSION}
PRETTY_NAME="MIXOS GO ${VERSION}"
HOME_URL="https://mixos.dev"
DOCUMENTATION_URL="https://docs.mixos.dev"
SUPPORT_URL="https://mixos.dev/support"
BUG_REPORT_URL="https://github.com/mixos/mixos-go/issues"
LOGO=mixos-logo
EOF

    # /etc/hostname
    echo "mixos" > etc/hostname
    
    # /etc/hosts
    cat > etc/hosts << 'EOF'
127.0.0.1   localhost
127.0.1.1   mixos
::1         localhost ip6-localhost ip6-loopback
ff02::1     ip6-allnodes
ff02::2     ip6-allrouters
EOF

    # /etc/passwd
    cat > etc/passwd << 'EOF'
root:x:0:0:root:/root:/bin/bash
bin:x:1:1:bin:/bin:/usr/bin/nologin
daemon:x:2:2:daemon:/:/usr/bin/nologin
mail:x:8:12:mail:/var/spool/mail:/usr/bin/nologin
ftp:x:14:11:ftp:/srv/ftp:/usr/bin/nologin
http:x:33:33:http:/srv/http:/usr/bin/nologin
nobody:x:65534:65534:Nobody:/:/usr/bin/nologin
dbus:x:81:81:System Message Bus:/:/usr/bin/nologin
systemd-journal:x:190:190:systemd Journal:/:/usr/bin/nologin
systemd-network:x:192:192:systemd Network Management:/:/usr/bin/nologin
systemd-resolve:x:193:193:systemd Resolver:/:/usr/bin/nologin
systemd-timesync:x:194:194:systemd Time Synchronization:/:/usr/bin/nologin
systemd-coredump:x:195:195:systemd Core Dumper:/:/usr/bin/nologin
EOF

    # /etc/shadow (root with no password for live boot)
    cat > etc/shadow << 'EOF'
root::19000:0:99999:7:::
bin:!*:19000::::::
daemon:!*:19000::::::
mail:!*:19000::::::
ftp:!*:19000::::::
http:!*:19000::::::
nobody:!*:19000::::::
dbus:!*:19000::::::
systemd-journal:!*:19000::::::
systemd-network:!*:19000::::::
systemd-resolve:!*:19000::::::
systemd-timesync:!*:19000::::::
systemd-coredump:!*:19000::::::
EOF
    chmod 600 etc/shadow

    # /etc/group
    cat > etc/group << 'EOF'
root:x:0:root
bin:x:1:root,bin,daemon
daemon:x:2:root,bin,daemon
sys:x:3:root,bin
adm:x:4:root,daemon
tty:x:5:
disk:x:6:root
lp:x:7:daemon
mem:x:8:
kmem:x:9:
wheel:x:10:root
ftp:x:11:
mail:x:12:
uucp:x:14:
log:x:19:root
utmp:x:20:
locate:x:21:
rfkill:x:24:
smmsp:x:25:
proc:x:26:
http:x:33:
games:x:50:
lock:x:54:
uuidd:x:68:
dbus:x:81:
network:x:90:
video:x:91:
audio:x:92:
optical:x:93:
floppy:x:94:
storage:x:95:
scanner:x:96:
input:x:97:
power:x:98:
nobody:x:65534:
users:x:100:
systemd-journal:x:190:
systemd-network:x:192:
systemd-resolve:x:193:
systemd-timesync:x:194:
systemd-coredump:x:195:
docker:x:999:
EOF

    # /etc/gshadow
    cat > etc/gshadow << 'EOF'
root:::root
wheel:::root
docker:::
EOF
    chmod 600 etc/gshadow

    # /etc/fstab (template - will be generated during install)
    cat > etc/fstab << 'EOF'
# /etc/fstab: static file system information
# <file system>  <mount point>  <type>  <options>  <dump>  <pass>

# Root filesystem (will be configured by installer)
# UUID=xxx  /  ext4  defaults  0  1

# Temporary filesystems
tmpfs  /tmp      tmpfs  defaults,noatime,mode=1777  0  0
tmpfs  /var/tmp  tmpfs  defaults,noatime,mode=1777  0  0
EOF

    # /etc/locale.conf
    echo "LANG=en_US.UTF-8" > etc/locale.conf
    
    # /etc/vconsole.conf
    echo "KEYMAP=us" > etc/vconsole.conf
    
    # /etc/localtime (symlink to UTC by default)
    ln -sf ../usr/share/zoneinfo/UTC etc/localtime
    
    # /etc/resolv.conf (will be managed by systemd-resolved)
    cat > etc/resolv.conf << 'EOF'
# Generated by MIXOS
# This file will be managed by systemd-resolved
nameserver 8.8.8.8
nameserver 8.8.4.4
EOF

    # /etc/nsswitch.conf
    cat > etc/nsswitch.conf << 'EOF'
passwd:     files systemd
group:      files systemd
shadow:     files
hosts:      files mymachines resolve [!UNAVAIL=return] dns myhostname
networks:   files
protocols:  files
services:   files
ethers:     files
rpc:        files
EOF

    log_success "Base system files created"
}

install_kernel_modules() {
    log_step "[3/10] Installing kernel modules..."
    
    cd "$ROOTFS_ROOT"
    
    if [ ! -d "$KERNEL_MODULES_SRC" ]; then
        log_warn "Kernel modules not found, skipping"
        return
    fi
    
    # Copy all kernel modules
    mkdir -p "lib/modules"
    cp -a "$KERNEL_MODULES_SRC" "lib/modules/"
    
    # Count modules
    local mod_count=$(find "lib/modules/${KERNEL_FULL_VERSION}" -name "*.ko*" | wc -l)
    local mod_size=$(du -sh "lib/modules/${KERNEL_FULL_VERSION}" | cut -f1)
    
    log_success "Installed $mod_count kernel modules ($mod_size)"
}

install_mixos_packages() {
    log_step "[4/10] Installing MIXOS packages..."
    
    cd "$ROOTFS_ROOT"
    
    # Install mix-cli
    if [ -f "$PKG_MIX_CLI" ]; then
        cp "$PKG_MIX_CLI" usr/local/bin/mix-cli
        chmod 755 usr/local/bin/mix-cli
        ln -sf mix-cli usr/local/bin/mix
        log_info "Installed: mix-cli"
    fi
    
    # Install mix-pkg
    if [ -f "$PKG_MIX_PKG" ]; then
        cp "$PKG_MIX_PKG" usr/local/bin/mix-pkg
        chmod 755 usr/local/bin/mix-pkg
        log_info "Installed: mix-pkg"
    fi
    
    # Install mix-agent (Python package)
    if [ -d "$PKG_MIX_AGENT" ]; then
        mkdir -p opt/mixos/agent
        cp -r "$PKG_MIX_AGENT"/* opt/mixos/agent/
        
        # Create wrapper script
        cat > usr/local/bin/mix-agent << 'EOF'
#!/bin/bash
export PYTHONPATH=/opt/mixos/agent:$PYTHONPATH
exec python3 -m mixos_agent "$@"
EOF
        chmod 755 usr/local/bin/mix-agent
        log_info "Installed: mix-agent"
    fi
    
    # Install mix-agent-early
    if [ -f "$PKG_MIX_AGENT_EARLY" ]; then
        cp "$PKG_MIX_AGENT_EARLY" usr/local/bin/mix-agent-early
        chmod 755 usr/local/bin/mix-agent-early
        log_info "Installed: mix-agent-early"
    fi
    
    # Install mix-installer
    if [ -f "$PKG_MIX_INSTALLER" ]; then
        cp "$PKG_MIX_INSTALLER" usr/local/bin/mix-installer
        chmod 755 usr/local/bin/mix-installer
        log_info "Installed: mix-installer"
    fi
    
    log_success "MIXOS packages installed"
}

install_ai_model() {
    log_step "[5/10] Installing AI model..."
    
    cd "$ROOTFS_ROOT"
    
    if [ -f "$AI_MODEL_FULL" ]; then
        cp "$AI_MODEL_FULL" opt/mixos/ai/model/
        local model_size=$(du -h "$AI_MODEL_FULL" | cut -f1)
        log_success "AI model installed ($model_size)"
    else
        log_warn "AI model not found: $AI_MODEL_FULL"
        log_warn "AI functionality will be limited"
        
        # Create placeholder
        echo "# AI model placeholder - download the actual model" > opt/mixos/ai/model/README
    fi
}

install_systemd_services() {
    log_step "[6/10] Installing systemd services..."
    
    cd "$ROOTFS_ROOT"
    
    # Copy services from configs directory
    local services_src="${CONFIGS_SRC_DIR}/systemd/services"
    if [ -d "$services_src" ]; then
        for service in "$services_src"/*.service; do
            if [ -f "$service" ]; then
                cp "$service" etc/systemd/system/
                log_info "Installed service: $(basename "$service")"
            fi
        done
    fi
    
    # Create mixos-agent.service if not exists
    if [ ! -f "etc/systemd/system/mixos-agent.service" ]; then
        cat > etc/systemd/system/mixos-agent.service << 'EOF'
[Unit]
Description=MIXOS AI Agent
Documentation=https://mixos.dev/docs/agent
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
Group=root
Environment=PYTHONPATH=/opt/mixos/agent
Environment=MIXOS_CONFIG=/etc/mixos
Environment=MIXOS_DATA=/var/lib/mixos
ExecStart=/usr/local/bin/mix-agent start --foreground
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
    fi
    
    # Create mixos-firstboot.service
    cat > etc/systemd/system/mixos-firstboot.service << 'EOF'
[Unit]
Description=MIXOS First Boot Setup
ConditionPathExists=!/var/lib/mixos/.firstboot-done
After=network.target

[Service]
Type=oneshot
ExecStart=/opt/mixos/scripts/firstboot.sh
ExecStartPost=/bin/touch /var/lib/mixos/.firstboot-done
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

    # Create mixos-installer.service (for live boot)
    cat > etc/systemd/system/mixos-installer.service << 'EOF'
[Unit]
Description=MIXOS Installer
ConditionPathExists=/run/mixos/installer
After=multi-user.target

[Service]
Type=simple
ExecStart=/usr/local/bin/mix-installer
StandardInput=tty
StandardOutput=tty
TTYPath=/dev/tty1
TTYReset=yes
TTYVHangup=yes

[Install]
WantedBy=multi-user.target
EOF

    # Enable default services (create symlinks)
    mkdir -p etc/systemd/system/multi-user.target.wants
    ln -sf ../mixos-agent.service etc/systemd/system/multi-user.target.wants/
    ln -sf ../mixos-firstboot.service etc/systemd/system/multi-user.target.wants/
    
    log_success "Systemd services installed"
}

install_overlay_files() {
    log_step "[7/10] Installing overlay files..."
    
    cd "$ROOTFS_ROOT"
    
    local overlay_src="${ROOTFS_SRC_DIR}/overlay"
    
    if [ -d "$overlay_src" ]; then
        # Copy overlay files preserving structure
        cp -a "$overlay_src"/* ./
        log_success "Overlay files installed"
    else
        log_warn "Overlay directory not found: $overlay_src"
    fi
}

install_configuration() {
    log_step "[8/10] Installing configuration files..."
    
    cd "$ROOTFS_ROOT"
    
    # /etc/mixos/agent.toml
    if [ ! -f "etc/mixos/agent.toml" ]; then
        cat > etc/mixos/agent.toml << 'EOF'
# MIXOS AI Agent Configuration

[agent]
name = "Mix Agent"
version = "1.0.0"
enabled = true
auto_start = true

[model]
path = "/opt/mixos/ai/model/mix-small-1.1b-q4_k_m.gguf"
context_length = 4096
temperature = 0.7
top_p = 0.95
top_k = 40
repeat_penalty = 1.1

[inference]
threads = 4
batch_size = 512
gpu_layers = 0

[memory]
max_conversation_history = 50
enable_long_term_memory = true
memory_db_path = "/var/lib/mixos/agent/memory.db"

[safety]
require_confirmation = ["install_package", "remove_package", "system_update", "disk_format"]
forbidden_commands = ["rm -rf /", "dd if=/dev/zero of=/dev/sd", "mkfs /dev/sd"]
max_execution_time = 300
enable_sandboxing = true

[api]
host = "127.0.0.1"
port = 8765
enable_cors = true

[logging]
level = "info"
path = "/var/log/mixos/agent.log"
max_size = "100MB"
rotation = "daily"
EOF
    fi
    
    # /etc/mixos/system.toml
    cat > etc/mixos/system.toml << EOF
# MIXOS System Configuration

[system]
version = "${VERSION}"
kernel = "${KERNEL_FULL_VERSION}"
arch = "x86_64"

[boot]
default_target = "multi-user.target"
timeout = 5

[network]
manager = "systemd-networkd"
dns = "systemd-resolved"

[packages]
manager = "mix-pkg"
repositories = ["/etc/mixos/repositories.toml"]
EOF

    # /etc/mixos/repositories.toml
    if [ ! -f "etc/mixos/repositories.toml" ]; then
        cat > etc/mixos/repositories.toml << 'EOF'
# MIXOS Package Repositories

[[repository]]
name = "core"
url = "https://repo.mixos.dev/core"
enabled = true
priority = 1

[[repository]]
name = "extra"
url = "https://repo.mixos.dev/extra"
enabled = true
priority = 2

[[repository]]
name = "community"
url = "https://repo.mixos.dev/community"
enabled = false
priority = 3
EOF
    fi
    
    # /etc/profile.d/mixos.sh
    cat > etc/profile.d/mixos.sh << 'EOF'
# MIXOS Environment Setup

export MIXOS_VERSION="1.0.0"
export MIXOS_CONFIG="/etc/mixos"
export MIXOS_DATA="/var/lib/mixos"

# Add MIXOS binaries to PATH
export PATH="/usr/local/bin:$PATH"

# Aliases
alias mix-status='mix status'
alias mix-update='mix update'

# Welcome message (only for interactive shells)
if [ -t 0 ]; then
    if [ -f /usr/share/mixos/welcome.txt ]; then
        cat /usr/share/mixos/welcome.txt
    fi
fi
EOF
    chmod 644 etc/profile.d/mixos.sh
    
    log_success "Configuration files installed"
}

install_scripts() {
    log_step "[9/10] Installing scripts..."
    
    cd "$ROOTFS_ROOT"
    
    # First boot script
    cat > opt/mixos/scripts/firstboot.sh << 'EOF'
#!/bin/bash
# MIXOS First Boot Script

echo "MIXOS First Boot Setup"

# Generate machine-id if not exists
if [ ! -f /etc/machine-id ] || [ ! -s /etc/machine-id ]; then
    systemd-machine-id-setup
fi

# Generate SSH host keys if not exists
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
    ssh-keygen -A 2>/dev/null || true
fi

# Update package database
if command -v mix-pkg &>/dev/null; then
    mix-pkg update 2>/dev/null || true
fi

# Initialize AI agent database
mkdir -p /var/lib/mixos/agent
touch /var/lib/mixos/agent/memory.db

echo "First boot setup complete"
EOF
    chmod 755 opt/mixos/scripts/firstboot.sh
    
    # Welcome message
    mkdir -p usr/share/mixos
    cat > usr/share/mixos/welcome.txt << 'EOF'

  ███╗   ███╗██╗██╗  ██╗ ██████╗ ███████╗     ██████╗  ██████╗ 
  ████╗ ████║██║╚██╗██╔╝██╔═══██╗██╔════╝    ██╔════╝ ██╔═══██╗
  ██╔████╔██║██║ ╚███╔╝ ██║   ██║███████╗    ██║  ███╗██║   ██║
  ██║╚██╔╝██║██║ ██╔██╗ ██║   ██║╚════██║    ██║   ██║██║   ██║
  ██║ ╚═╝ ██║██║██╔╝ ██╗╚██████╔╝███████║    ╚██████╔╝╚██████╔╝
  ╚═╝     ╚═╝╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝     ╚═════╝  ╚═════╝ 
                                                                
  AI-Powered Operating System for Developers
  
  Quick Start:
    mix status          - Show system status
    mix agent chat      - Chat with AI assistant
    mix install <pkg>   - Install a package
    mix help            - Show all commands

EOF

    log_success "Scripts installed"
}

finalize() {
    log_step "[10/10] Finalizing..."
    
    cd "$ROOTFS_ROOT"
    
    # Create /etc/ld.so.conf.d directory
    mkdir -p etc/ld.so.conf.d
    
    # Create package manifest
    cat > var/lib/mixos/packages.manifest << EOF
# MIXOS Package Manifest
# Generated: $(date -Iseconds)

mixos-base ${VERSION}
linux-kernel ${KERNEL_FULL_VERSION}
mix-cli ${VERSION}
mix-pkg ${VERSION}
mix-agent ${VERSION}
mix-installer ${VERSION}
EOF

    # Set proper permissions
    chmod 755 bin sbin usr/bin usr/sbin usr/local/bin
    chmod 755 opt/mixos opt/mixos/scripts opt/mixos/ai
    
    # Create empty log files
    touch var/log/mixos/agent.log 2>/dev/null || mkdir -p var/log/mixos
    
    log_success "Rootfs finalized"
}

show_summary() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " Root Filesystem Build Complete"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Version:     ${VERSION}"
    echo " Kernel:      ${KERNEL_FULL_VERSION}"
    echo " Output:      ${ROOTFS_ROOT}"
    echo ""
    
    if [ -d "$ROOTFS_ROOT" ]; then
        local total_size=$(du -sh "$ROOTFS_ROOT" | cut -f1)
        echo " Total Size:  $total_size"
        echo ""
        echo " Directory Sizes:"
        du -sh "$ROOTFS_ROOT"/{bin,lib,usr,opt,etc,var} 2>/dev/null | while read size dir; do
            printf "   %-20s %s\n" "$(basename "$dir"):" "$size"
        done
    fi
    
    echo ""
    echo " Next step: Run 'make rootfs-squash' to create squashfs image"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo " MIXOS GO - Root Filesystem Build"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo " Version:         ${VERSION}"
    echo " Kernel:          ${KERNEL_FULL_VERSION}"
    echo " Output:          ${ROOTFS_ROOT}"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    
    check_prerequisites
    create_directory_structure
    create_base_system
    install_kernel_modules
    install_mixos_packages
    install_ai_model
    install_systemd_services
    install_overlay_files
    install_configuration
    install_scripts
    finalize
    show_summary
}

# Run main
main "$@"
