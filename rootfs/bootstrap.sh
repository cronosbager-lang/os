#!/bin/bash
# MIXOS RootFS Bootstrap Script
# Creates the root filesystem for MIXOS GO

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$ROOT_DIR/build/rootfs"
WORK_DIR="$BUILD_DIR/work"

ARCH="${ARCH:-x86_64}"
PROFILE="${PROFILE:-developer}"

echo "=========================================="
echo "MIXOS RootFS Bootstrap"
echo "Profile: $PROFILE"
echo "Architecture: $ARCH"
echo "=========================================="

# Create work directory
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

# Step 1: Create base directory structure
echo "[1/8] Creating directory structure..."
mkdir -p {bin,sbin,lib,lib64,usr/{bin,sbin,lib,lib64,share,local/bin},etc,var/{log,lib,cache,run},tmp,root,home,opt,mnt,proc,sys,dev,run}

# Step 2: Install base system (using debootstrap or similar)
echo "[2/8] Installing base system..."
# In production, this would use debootstrap, pacstrap, or similar
# For now, we create a minimal structure

# Create essential files
cat > etc/os-release << 'EOF'
NAME="MIXOS GO"
VERSION="1.0"
ID=mixos
ID_LIKE=arch
VERSION_ID=1.0
PRETTY_NAME="MIXOS GO 1.0"
HOME_URL="https://mixos.dev"
EOF

cat > etc/hostname << 'EOF'
mixos
EOF

cat > etc/hosts << 'EOF'
127.0.0.1   localhost
127.0.1.1   mixos
::1         localhost ip6-localhost ip6-loopback
EOF

# Step 3: Copy overlay files
echo "[3/8] Applying overlay..."
if [ -d "$SCRIPT_DIR/overlay" ]; then
    cp -a "$SCRIPT_DIR/overlay/"* "$WORK_DIR/"
fi

# Step 4: Install MIXOS components
echo "[4/8] Installing MIXOS components..."

# Copy mix-cli
if [ -f "$ROOT_DIR/build/mix-cli/mix-cli" ]; then
    cp "$ROOT_DIR/build/mix-cli/mix-cli" "$WORK_DIR/usr/local/bin/"
fi

# Copy mix-pkg
if [ -f "$ROOT_DIR/build/mix-pkg/mix-pkg" ]; then
    cp "$ROOT_DIR/build/mix-pkg/mix-pkg" "$WORK_DIR/usr/local/bin/"
fi

# Copy mix-installer
if [ -f "$ROOT_DIR/build/mix-installer/mix-installer" ]; then
    cp "$ROOT_DIR/build/mix-installer/mix-installer" "$WORK_DIR/usr/local/bin/"
fi

# Step 5: Install AI agent
echo "[5/8] Installing AI agent..."
mkdir -p "$WORK_DIR/opt/mixos/ai/model"
mkdir -p "$WORK_DIR/var/lib/mixos/agent"

# Copy mix-agent Python package
if [ -d "$ROOT_DIR/mix-agent" ]; then
    mkdir -p "$WORK_DIR/opt/mixos/agent"
    cp -r "$ROOT_DIR/mix-agent/mixos_agent" "$WORK_DIR/opt/mixos/agent/"
    cp "$ROOT_DIR/mix-agent/requirements.txt" "$WORK_DIR/opt/mixos/agent/"
fi

# Step 6: Configure systemd services
echo "[6/8] Configuring services..."
mkdir -p "$WORK_DIR/etc/systemd/system"

cat > "$WORK_DIR/etc/systemd/system/mixos-agent.service" << 'EOF'
[Unit]
Description=MIXOS AI Agent
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
ExecStart=/usr/bin/python3 -m mixos_agent start --foreground
Restart=always
RestartSec=5
Environment=PYTHONPATH=/opt/mixos/agent

[Install]
WantedBy=multi-user.target
EOF

# Step 7: Set permissions
echo "[7/8] Setting permissions..."
chmod 755 "$WORK_DIR/usr/local/bin/"* 2>/dev/null || true
chmod 644 "$WORK_DIR/etc/systemd/system/"*.service 2>/dev/null || true

# Step 8: Create package list
echo "[8/8] Creating package manifest..."
cat > "$WORK_DIR/var/lib/mixos/packages.list" << 'EOF'
mixos-base
linux-kernel
systemd
docker
mix-cli
mix-pkg
mix-agent
EOF

echo "=========================================="
echo "RootFS bootstrap complete!"
echo "Output: $WORK_DIR"
echo "=========================================="
