#!/bin/bash
# MIXOS Environment Setup

# Add MIXOS binaries to PATH
export PATH="/usr/local/bin:$PATH"

# MIXOS configuration
export MIXOS_CONFIG="/etc/mixos"
export MIXOS_DATA="/var/lib/mixos"

# AI Agent
export MIXOS_AGENT_CONFIG="/etc/mixos/agent.toml"

# Aliases
alias mix-status='mix status'
alias mix-update='mix update && mix upgrade'
alias ai='mix agent chat'

# Welcome message (only for interactive shells)
if [[ $- == *i* ]]; then
    if [ -f /usr/share/mixos/welcome.txt ]; then
        cat /usr/share/mixos/welcome.txt
    fi
fi
