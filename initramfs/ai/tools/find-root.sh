#!/bin/sh
# MIXOS Root Filesystem Finder
# Used by AI agent to locate root filesystem

echo "=== Searching for Root Filesystem ==="
echo ""

# Check kernel command line for root parameter
echo "## Kernel root parameter"
root_param=$(cat /proc/cmdline | tr ' ' '\n' | grep "^root=" | cut -d= -f2-)
if [ -n "$root_param" ]; then
    echo "Specified: $root_param"
else
    echo "Not specified in kernel parameters"
fi
echo ""

# Search by label
echo "## By Label"
if [ -d /dev/disk/by-label ]; then
    for label in /dev/disk/by-label/*; do
        if [ -L "$label" ]; then
            name=$(basename "$label")
            target=$(readlink -f "$label")
            echo "$name -> $target"
            
            # Check for MIXOS labels
            case "$name" in
                MIXOS_ROOT|mixos_root|root)
                    echo "  *** Likely root filesystem ***"
                    ;;
            esac
        fi
    done
else
    echo "No labels found"
fi
echo ""

# Search by UUID
echo "## By UUID"
if [ -d /dev/disk/by-uuid ]; then
    ls /dev/disk/by-uuid/ | head -10
    total=$(ls /dev/disk/by-uuid/ 2>/dev/null | wc -l)
    if [ "$total" -gt 10 ]; then
        echo "... ($total total UUIDs)"
    fi
fi
echo ""

# Check common devices
echo "## Common Devices"
for dev in /dev/sda1 /dev/sda2 /dev/sda3 \
           /dev/vda1 /dev/vda2 /dev/vda3 \
           /dev/nvme0n1p1 /dev/nvme0n1p2 /dev/nvme0n1p3; do
    if [ -b "$dev" ]; then
        fstype=""
        label=""
        
        if command -v blkid >/dev/null 2>&1; then
            info=$(blkid "$dev" 2>/dev/null)
            fstype=$(echo "$info" | grep -o 'TYPE="[^"]*"' | cut -d'"' -f2)
            label=$(echo "$info" | grep -o 'LABEL="[^"]*"' | cut -d'"' -f2)
        fi
        
        echo "$dev: $fstype ${label:+($label)}"
        
        # Check if it looks like a root filesystem
        if [ "$fstype" = "ext4" ] || [ "$fstype" = "btrfs" ] || [ "$fstype" = "xfs" ]; then
            # Try to mount and check for /etc
            mkdir -p /tmp/rootcheck 2>/dev/null
            if mount -o ro "$dev" /tmp/rootcheck 2>/dev/null; then
                if [ -d /tmp/rootcheck/etc ] && [ -d /tmp/rootcheck/bin ]; then
                    echo "  *** Valid root filesystem ***"
                    
                    # Check for MIXOS
                    if [ -f /tmp/rootcheck/etc/mixos-release ]; then
                        echo "  *** MIXOS installation found ***"
                        cat /tmp/rootcheck/etc/mixos-release
                    fi
                fi
                umount /tmp/rootcheck 2>/dev/null
            fi
        fi
    fi
done
echo ""

# Recommendations
echo "## Recommendations"
if [ -n "$root_param" ]; then
    if [ -b "$root_param" ]; then
        echo "Specified root device exists: $root_param"
    else
        echo "WARNING: Specified root device not found: $root_param"
        echo "Consider using one of the detected devices above"
    fi
else
    echo "No root specified. Suggested boot parameters:"
    
    # Find best candidate
    for dev in /dev/disk/by-label/MIXOS_ROOT \
               /dev/nvme0n1p2 /dev/sda2 /dev/vda2; do
        if [ -e "$dev" ]; then
            target=$(readlink -f "$dev" 2>/dev/null || echo "$dev")
            echo "  root=$target"
            break
        fi
    done
fi
echo ""

echo "=== End Root Search ==="
