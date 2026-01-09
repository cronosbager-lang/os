#!/bin/bash
# MIXOS GO First Boot Script
# Runs once on first boot to configure the system

set -e

LOG_FILE="/var/log/mixos/firstboot.log"
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "MIXOS GO First Boot Configuration Starting"

# Generate machine ID if not exists
if [ ! -f /etc/machine-id ] || [ ! -s /etc/machine-id ]; then
    log "Generating machine ID..."
    systemd-machine-id-setup
fi

# Generate SSH host keys if not exist
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
    log "Generating SSH host keys..."
    ssh-keygen -A
fi

# Initialize package database
if command -v mix-pkg &> /dev/null; then
    log "Initializing package database..."
    mix-pkg update 2>/dev/null || true
fi

# Setup AI agent
if [ -d /opt/mixos/agent ]; then
    log "Setting up AI agent..."
    
    # Create agent directories
    mkdir -p /var/lib/mixos/agent
    mkdir -p /var/log/mixos
    
    # Set permissions
    chmod 755 /var/lib/mixos/agent
    chmod 755 /var/log/mixos
fi

# Enable essential services
log "Enabling services..."
systemctl enable systemd-networkd 2>/dev/null || true
systemctl enable systemd-resolved 2>/dev/null || true
systemctl enable docker 2>/dev/null || true
systemctl enable mixos-agent 2>/dev/null || true

# Create welcome message
cat > /etc/motd << 'EOF'

  __  __ _____  _____  ____   _____    _____  ____  
 |  \/  |_   _|/ ____|/ __ \ / ____|  / ____|/ __ \ 
 | \  / | | | | (___ | |  | | (___   | |  __| |  | |
 | |\/| | | |  \___ \| |  | |\___ \  | | |_ | |  | |
 | |  | |_| |_ ____) | |__| |____) | | |__| | |__| |
 |_|  |_|_____|_____/ \____/|_____/   \_____|\____/ 
                                                    
  AI-Powered Operating System for Developers

  Quick Start:
    mix status          - Check system status
    mix agent chat      - Chat with AI assistant
    mix --help          - Show all commands

  Documentation: https://mixos.dev/docs

EOF

# Sync filesystem
sync

log "First boot configuration complete"
