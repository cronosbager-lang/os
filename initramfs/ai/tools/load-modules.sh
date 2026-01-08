#!/bin/sh
# MIXOS Module Loader
# Used by AI agent to load kernel modules

usage() {
    echo "Usage: $0 [options] [module...]"
    echo ""
    echo "Options:"
    echo "  -a, --auto     Auto-detect and load required modules"
    echo "  -l, --list     List available modules"
    echo "  -s, --status   Show loaded modules"
    echo "  -h, --help     Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 nvme ext4           # Load specific modules"
    echo "  $0 --auto              # Auto-detect and load"
    echo "  $0 --list | grep nvme  # Search for modules"
}

list_modules() {
    echo "=== Available Modules ==="
    if [ -d /lib/modules ]; then
        kernel=$(uname -r)
        if [ -d "/lib/modules/$kernel" ]; then
            find "/lib/modules/$kernel" -name "*.ko*" -exec basename {} \; | \
                sed 's/\.ko.*$//' | sort -u
        else
            echo "No modules found for kernel $kernel"
        fi
    else
        echo "Module directory not found"
    fi
}

show_status() {
    echo "=== Loaded Modules ==="
    if [ -f /proc/modules ]; then
        awk '{print $1, $3}' /proc/modules | column -t
    else
        echo "Cannot read /proc/modules"
    fi
}

load_module() {
    module="$1"
    
    # Check if already loaded
    if grep -q "^$module " /proc/modules 2>/dev/null; then
        echo "Module $module already loaded"
        return 0
    fi
    
    # Try to load
    echo -n "Loading $module... "
    if modprobe "$module" 2>/dev/null; then
        echo "OK"
        return 0
    else
        echo "FAILED"
        return 1
    fi
}

auto_detect() {
    echo "=== Auto-detecting Required Modules ==="
    
    loaded=0
    failed=0
    
    # Storage modules based on detected hardware
    echo ""
    echo "## Storage"
    
    # Check for NVMe
    if [ -d /sys/class/nvme ] || ls /dev/nvme* >/dev/null 2>&1; then
        load_module nvme && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module nvme_core && loaded=$((loaded+1)) || failed=$((failed+1))
    fi
    
    # Check for SATA/AHCI
    if grep -q "AHCI" /proc/scsi/scsi 2>/dev/null || \
       ls /sys/class/scsi_host/host*/proc_name 2>/dev/null | xargs cat 2>/dev/null | grep -q ahci; then
        load_module ahci && loaded=$((loaded+1)) || failed=$((failed+1))
    fi
    
    # Always load sd_mod for SCSI disks
    load_module sd_mod && loaded=$((loaded+1)) || failed=$((failed+1))
    
    # VirtIO (for VMs)
    if grep -q "QEMU\|KVM\|VirtualBox\|VMware" /sys/class/dmi/id/sys_vendor 2>/dev/null; then
        load_module virtio_blk && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module virtio_scsi && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module virtio_pci && loaded=$((loaded+1)) || failed=$((failed+1))
    fi
    
    # USB storage
    if [ -d /sys/bus/usb ]; then
        load_module usb_storage && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module xhci_hcd && loaded=$((loaded+1)) || failed=$((failed+1))
    fi
    
    # Filesystem modules based on detected partitions
    echo ""
    echo "## Filesystems"
    
    if command -v blkid >/dev/null 2>&1; then
        for fstype in $(blkid -o value -s TYPE 2>/dev/null | sort -u); do
            case "$fstype" in
                ext4|ext3|ext2)
                    load_module ext4 && loaded=$((loaded+1)) || failed=$((failed+1))
                    ;;
                btrfs)
                    load_module btrfs && loaded=$((loaded+1)) || failed=$((failed+1))
                    ;;
                xfs)
                    load_module xfs && loaded=$((loaded+1)) || failed=$((failed+1))
                    ;;
                vfat|fat32|fat16)
                    load_module vfat && loaded=$((loaded+1)) || failed=$((failed+1))
                    load_module fat && loaded=$((loaded+1)) || failed=$((failed+1))
                    ;;
                squashfs)
                    load_module squashfs && loaded=$((loaded+1)) || failed=$((failed+1))
                    ;;
            esac
        done
    else
        # Load common filesystems
        load_module ext4 && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module vfat && loaded=$((loaded+1)) || failed=$((failed+1))
        load_module squashfs && loaded=$((loaded+1)) || failed=$((failed+1))
    fi
    
    # Always load overlay and loop for live boot
    load_module overlay && loaded=$((loaded+1)) || failed=$((failed+1))
    load_module loop && loaded=$((loaded+1)) || failed=$((failed+1))
    
    # Input modules
    echo ""
    echo "## Input"
    load_module hid_generic && loaded=$((loaded+1)) || failed=$((failed+1))
    load_module usbhid && loaded=$((loaded+1)) || failed=$((failed+1))
    
    echo ""
    echo "=== Summary ==="
    echo "Loaded: $loaded modules"
    echo "Failed: $failed modules"
    
    # Settle udev
    if command -v udevadm >/dev/null 2>&1; then
        echo ""
        echo "Waiting for devices to settle..."
        udevadm settle --timeout=5
    fi
}

# Main
case "$1" in
    -a|--auto)
        auto_detect
        ;;
    -l|--list)
        list_modules
        ;;
    -s|--status)
        show_status
        ;;
    -h|--help|"")
        usage
        ;;
    *)
        # Load specified modules
        for mod in "$@"; do
            load_module "$mod"
        done
        ;;
esac
