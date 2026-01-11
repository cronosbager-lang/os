#!/bin/sh
# MixOS Agent Runner
# Starts the Python agent with IPC integration

set -e

AGENT_DIR="/agent"
MIXOS_AGENT_DIR="/opt/mixos/agent"

# Check if running in MixOS environment
if [ -d "$MIXOS_AGENT_DIR" ]; then
    cd "$MIXOS_AGENT_DIR"
else
    cd "$AGENT_DIR"
fi

# Set environment
export PYTHONPATH="${PYTHONPATH}:${PWD}"
export MIXOS_IPC_SOCKET="/run/mixos/ipc.sock"
export MIXOS_STORE_PATH="/store"
export MIXOS_LOG_LEVEL="${MIXOS_LOG_LEVEL:-INFO}"

# Wait for IPC socket
echo "[agent] Waiting for IPC broker..."
for i in $(seq 1 30); do
    if [ -S "$MIXOS_IPC_SOCKET" ]; then
        break
    fi
    sleep 1
done

if [ ! -S "$MIXOS_IPC_SOCKET" ]; then
    echo "[agent] ERROR: IPC broker not available"
    exit 1
fi

echo "[agent] Starting MixOS Agent..."

# Run agent
exec python3 -m mixos_agent "$@"
