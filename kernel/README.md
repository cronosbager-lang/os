# MIXOS Kernel

Custom Linux kernel configuration optimized for MIXOS GO.

## Configurations

- `mixos-default.config` - Default configuration with full hardware support
- `mixos-minimal.config` - Minimal configuration for VMs
- `mixos-dev.config` - Development configuration with debug symbols

## Building

```bash
# Download kernel source
make download KERNEL_VERSION=6.6.10

# Configure
make menuconfig

# Build
make build

# Install modules to staging
make modules_install DESTDIR=../build/modules
```

## Key Features

- Optimized for containers (cgroups v2, namespaces)
- NVMe and SATA support
- Network drivers (Intel, Realtek, etc.)
- Filesystem support (ext4, btrfs, squashfs, overlay)
- Minimal unnecessary drivers removed
