# MixOS Package Format Specification

## Overview

MixOS packages use a TOML-based manifest format (`package.toml`) that defines package metadata, dependencies, build instructions, and installation hooks.

## Package Structure

```
<package-name>/
├── package.toml          # Package manifest
├── build.sh              # Build script (optional)
├── install.sh            # Install script (optional)
├── patches/              # Patches directory (optional)
│   └── *.patch
├── files/                # Additional files (optional)
│   └── ...
└── src/                  # Source files (optional)
    └── ...
```

## Package Manifest (package.toml)

### Basic Example

```toml
[package]
name = "hello"
version = "1.0.0"
description = "A simple hello world program"
license = "MIT"
homepage = "https://example.com/hello"

[source]
url = "https://example.com/hello-1.0.0.tar.gz"
sha256 = "abc123..."

[dependencies]
libc = ">=1.0"

[build]
type = "make"
```

### Full Specification

```toml
# =============================================================================
# Package Metadata
# =============================================================================

[package]
# Required: Package name (lowercase, alphanumeric, hyphens allowed)
name = "example-package"

# Required: Semantic version
version = "1.2.3"

# Required: Short description
description = "An example package for MixOS"

# Optional: Long description
long_description = """
This is a longer description that can span
multiple lines and provide more detail about
the package.
"""

# Optional: License identifier (SPDX)
license = "MIT"

# Optional: Package homepage
homepage = "https://example.com"

# Optional: Repository URL
repository = "https://github.com/example/package"

# Optional: Bug tracker URL
bugs = "https://github.com/example/package/issues"

# Optional: Package maintainers
maintainers = [
    "John Doe <john@example.com>",
    "Jane Smith <jane@example.com>",
]

# Optional: Package keywords for search
keywords = ["example", "demo", "utility"]

# Optional: Package categories
categories = ["utilities", "development"]

# =============================================================================
# Source Configuration
# =============================================================================

[source]
# Source URL (supports http, https, git, local)
url = "https://example.com/package-1.2.3.tar.gz"

# SHA256 checksum of the source archive
sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

# Optional: Source type (auto-detected if not specified)
# Values: tarball, zip, git, local
type = "tarball"

# Optional: Git-specific options
[source.git]
branch = "main"
tag = "v1.2.3"
commit = "abc123"
depth = 1

# Optional: Patches to apply
[[source.patches]]
file = "patches/fix-build.patch"
strip = 1

[[source.patches]]
url = "https://example.com/patches/security-fix.patch"
sha256 = "..."

# =============================================================================
# Dependencies
# =============================================================================

[dependencies]
# Runtime dependencies
# Format: package = "version-constraint"
# Constraints: =, !=, >, <, >=, <=, ~> (pessimistic)
libc = ">=1.0"
openssl = ">=1.1,<2.0"
zlib = "~>1.2"

[dependencies.build]
# Build-time only dependencies
gcc = ">=10.0"
make = "*"
cmake = ">=3.20"

[dependencies.optional]
# Optional dependencies
docs = { package = "sphinx", version = ">=4.0" }
gui = { package = "gtk4", version = ">=4.0" }

[dependencies.conflicts]
# Packages that conflict with this one
old-package = "*"

# =============================================================================
# Build Configuration
# =============================================================================

[build]
# Build system type
# Values: make, cmake, meson, autotools, cargo, go, custom
type = "make"

# Build directory (relative to source)
dir = "."

# Number of parallel jobs (0 = auto)
jobs = 0

# Environment variables for build
[build.env]
CFLAGS = "-O2 -pipe"
LDFLAGS = "-Wl,-O1"

# Configure options (for autotools/cmake)
[build.configure]
args = ["--prefix=/usr", "--enable-shared"]

# Make options
[build.make]
targets = ["all"]
install_targets = ["install"]
args = ["-j4"]

# CMake options
[build.cmake]
generator = "Ninja"
build_type = "Release"
args = ["-DBUILD_TESTS=OFF"]

# Meson options
[build.meson]
build_type = "release"
args = []

# Cargo options (for Rust)
[build.cargo]
features = ["default", "extra"]
args = ["--release"]

# Go options
[build.go]
ldflags = ["-s", "-w"]
tags = []

# Custom build script
[build.custom]
configure = "scripts/configure.sh"
build = "scripts/build.sh"
install = "scripts/install.sh"

# =============================================================================
# Installation Configuration
# =============================================================================

[install]
# Installation prefix
prefix = "/usr"

# Files to install (glob patterns)
files = [
    { src = "bin/*", dest = "bin/", mode = "755" },
    { src = "lib/*.so*", dest = "lib/", mode = "644" },
    { src = "include/*", dest = "include/", mode = "644" },
    { src = "share/*", dest = "share/", mode = "644" },
]

# Directories to create
directories = [
    { path = "etc/example", mode = "755" },
    { path = "var/lib/example", mode = "750" },
]

# Symlinks to create
symlinks = [
    { src = "bin/example", dest = "bin/ex" },
]

# =============================================================================
# Hooks
# =============================================================================

[hooks]
# Pre-install hook
pre_install = """
#!/bin/sh
echo "Preparing to install..."
"""

# Post-install hook
post_install = """
#!/bin/sh
echo "Installation complete!"
ldconfig
"""

# Pre-remove hook
pre_remove = """
#!/bin/sh
echo "Preparing to remove..."
"""

# Post-remove hook
post_remove = """
#!/bin/sh
echo "Removal complete!"
"""

# Pre-upgrade hook
pre_upgrade = """
#!/bin/sh
echo "Preparing to upgrade from $1 to $2..."
"""

# Post-upgrade hook
post_upgrade = """
#!/bin/sh
echo "Upgrade complete!"
"""

# =============================================================================
# Testing
# =============================================================================

[test]
# Enable tests
enabled = true

# Test command
command = "make test"

# Test timeout in seconds
timeout = 300

# Test dependencies
dependencies = ["check", "valgrind"]

# =============================================================================
# Provides and Replaces
# =============================================================================

[provides]
# Virtual packages this package provides
virtual = ["mail-transport-agent"]

# Specific versions provided
packages = [
    { name = "libfoo", version = "1.0" },
]

[replaces]
# Packages this one replaces
packages = ["old-example"]

# =============================================================================
# Platform-Specific Configuration
# =============================================================================

[platform.linux]
dependencies = { systemd = ">=250" }

[platform.linux.build.env]
CFLAGS = "-O2 -pipe -fPIC"

[platform.darwin]
dependencies = { launchd = "*" }

# =============================================================================
# Architecture-Specific Configuration
# =============================================================================

[arch.x86_64]
build.env.CFLAGS = "-O2 -march=x86-64"

[arch.aarch64]
build.env.CFLAGS = "-O2 -march=armv8-a"

# =============================================================================
# Metadata for Store
# =============================================================================

[store]
# Content hash (computed automatically)
# hash = "sha256:..."

# Store path (computed automatically)
# path = "/store/packages/<hash>"

# Output hash mode
# Values: flat, recursive
output_hash_mode = "recursive"
```

## Version Constraints

| Constraint | Meaning |
|------------|---------|
| `*` | Any version |
| `=1.0.0` | Exactly version 1.0.0 |
| `!=1.0.0` | Any version except 1.0.0 |
| `>1.0.0` | Greater than 1.0.0 |
| `<1.0.0` | Less than 1.0.0 |
| `>=1.0.0` | Greater than or equal to 1.0.0 |
| `<=1.0.0` | Less than or equal to 1.0.0 |
| `>=1.0,<2.0` | Range: 1.0.0 to 1.x.x |
| `~>1.2` | Pessimistic: >=1.2.0, <2.0.0 |
| `~>1.2.3` | Pessimistic: >=1.2.3, <1.3.0 |

## Build Types

### Make
```toml
[build]
type = "make"
[build.make]
targets = ["all"]
install_targets = ["install", "DESTDIR=$out"]
```

### CMake
```toml
[build]
type = "cmake"
[build.cmake]
generator = "Ninja"
build_type = "Release"
args = ["-DBUILD_SHARED_LIBS=ON"]
```

### Meson
```toml
[build]
type = "meson"
[build.meson]
build_type = "release"
```

### Autotools
```toml
[build]
type = "autotools"
[build.configure]
args = ["--prefix=/usr", "--sysconfdir=/etc"]
```

### Cargo (Rust)
```toml
[build]
type = "cargo"
[build.cargo]
features = ["default"]
args = ["--release"]
```

### Go
```toml
[build]
type = "go"
[build.go]
ldflags = ["-s", "-w"]
```

### Custom
```toml
[build]
type = "custom"
[build.custom]
configure = "./configure.sh"
build = "./build.sh"
install = "./install.sh $out"
```

## Store Integration

Packages are stored in the content-addressable store:

```
/store/packages/<sha256-hash>/
├── info.json           # Package metadata
├── content/            # Package contents
│   ├── bin/
│   ├── lib/
│   ├── include/
│   └── share/
└── manifest.json       # File manifest with hashes
```

## Example Packages

### Simple C Program
```toml
[package]
name = "hello"
version = "1.0.0"
description = "Hello World"
license = "MIT"

[source]
url = "https://example.com/hello-1.0.0.tar.gz"
sha256 = "..."

[dependencies]
libc = ">=1.0"

[build]
type = "make"
```

### Rust Application
```toml
[package]
name = "ripgrep"
version = "13.0.0"
description = "Fast grep alternative"
license = "MIT"

[source]
url = "https://github.com/BurntSushi/ripgrep/archive/13.0.0.tar.gz"
sha256 = "..."

[dependencies.build]
rust = ">=1.70"

[build]
type = "cargo"
[build.cargo]
features = ["pcre2"]
```

### Python Package
```toml
[package]
name = "python-requests"
version = "2.31.0"
description = "HTTP library for Python"
license = "Apache-2.0"

[source]
url = "https://pypi.org/packages/source/r/requests/requests-2.31.0.tar.gz"
sha256 = "..."

[dependencies]
python = ">=3.8"
python-urllib3 = ">=1.21"
python-certifi = ">=2017.4.17"

[build]
type = "custom"
[build.custom]
build = "python setup.py build"
install = "python setup.py install --prefix=$out"
```
