#!/bin/sh
# MIXOS Hardware Detection Tool
# Used by AI agent for hardware analysis

echo "=== MIXOS Hardware Detection ==="
echo ""

# CPU Information
echo "## CPU"
if [ -f /proc/cpuinfo ]; then
    grep -m1 "model name" /proc/cpuinfo | cut -d: -f2 | sed 's/^ //'
    grep -m1 "vendor_id" /proc/cpuinfo | cut -d: -f2 | sed 's/^ //'
    echo "Cores: $(grep -c "^processor" /proc/cpuinfo)"
fi
echo ""

# Memory Information
echo "## Memory"
if [ -f /proc/meminfo ]; then
    grep "MemTotal" /proc/meminfo
    grep "MemAvailable" /proc/meminfo
fi
echo ""

# Block Devices
echo "## Block Devices"
if command -v lsblk >/dev/null 2>&1; then
    lsblk -o NAME,SIZE,TYPE,FSTYPE,LABEL,MOUNTPOINT 2>/dev/null
else
    ls -la /sys/block/ 2>/dev/null | grep -v "^total"
fi
echo ""

# Storage Controllers
echo "## Storage Controllers"
for controller in /sys/class/block/*/device; do
    if [ -d "$controller" ]; then
        dev=$(dirname "$controller" | xargs basename)
        if [ -f "$controller/vendor" ]; then
            vendor=$(cat "$controller/vendor" 2>/dev/null | tr -d ' ')
            model=$(cat "$controller/model" 2>/dev/null | tr -d ' ')
            echo "$dev: $vendor $model"
        fi
    fi
done
echo ""

# PCI Devices (if available)
echo "## PCI Devices"
if [ -d /sys/bus/pci/devices ]; then
    for dev in /sys/bus/pci/devices/*; do
        if [ -f "$dev/class" ]; then
            class=$(cat "$dev/class" 2>/dev/null)
            # Storage controllers (0x01xxxx)
            case "$class" in
                0x01*)
                    vendor=$(cat "$dev/vendor" 2>/dev/null)
                    device=$(cat "$dev/device" 2>/dev/null)
                    echo "Storage: $vendor:$device"
                    ;;
                0x02*)
                    vendor=$(cat "$dev/vendor" 2>/dev/null)
                    device=$(cat "$dev/device" 2>/dev/null)
                    echo "Network: $vendor:$device"
                    ;;
                0x03*)
                    vendor=$(cat "$dev/vendor" 2>/dev/null)
                    device=$(cat "$dev/device" 2>/dev/null)
                    echo "Graphics: $vendor:$device"
                    ;;
            esac
        fi
    done
fi
echo ""

# Network Interfaces
echo "## Network Interfaces"
for iface in /sys/class/net/*; do
    if [ -d "$iface" ]; then
        name=$(basename "$iface")
        if [ "$name" != "lo" ]; then
            mac=$(cat "$iface/address" 2>/dev/null)
            state=$(cat "$iface/operstate" 2>/dev/null)
            echo "$name: $mac ($state)"
        fi
    fi
done
echo ""

# Loaded Modules
echo "## Loaded Modules"
if [ -f /proc/modules ]; then
    cut -d' ' -f1 /proc/modules | head -20
    total=$(wc -l < /proc/modules)
    echo "... ($total total modules)"
fi
echo ""

# Boot Mode
echo "## Boot Mode"
if [ -d /sys/firmware/efi ]; then
    echo "UEFI"
else
    echo "Legacy BIOS"
fi
echo ""

# Kernel Command Line
echo "## Kernel Parameters"
cat /proc/cmdline 2>/dev/null
echo ""

echo "=== End Hardware Detection ==="
