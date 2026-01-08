# MIXOS Kernel Build
KERNEL_VERSION ?= 6.6.10
KERNEL_MAJOR := $(shell echo $(KERNEL_VERSION) | cut -d. -f1)
KERNEL_URL := https://cdn.kernel.org/pub/linux/kernel/v$(KERNEL_MAJOR).x/linux-$(KERNEL_VERSION).tar.xz
KERNEL_DIR := linux-$(KERNEL_VERSION)
CONFIG ?= mixos-default

BUILD_DIR := $(CURDIR)/../build/kernel
MODULES_DIR := $(BUILD_DIR)/modules
NPROC := $(shell nproc)

.PHONY: all download extract configure build modules_install clean

all: build

download:
	@echo "==> Downloading Linux kernel $(KERNEL_VERSION)..."
	@if [ ! -f "linux-$(KERNEL_VERSION).tar.xz" ]; then \
		curl -L -o linux-$(KERNEL_VERSION).tar.xz $(KERNEL_URL); \
	fi

extract: download
	@echo "==> Extracting kernel source..."
	@if [ ! -d "$(KERNEL_DIR)" ]; then \
		tar xf linux-$(KERNEL_VERSION).tar.xz; \
	fi

configure: extract
	@echo "==> Configuring kernel with $(CONFIG).config..."
	@mkdir -p $(BUILD_DIR)
	@cp config/$(CONFIG).config $(KERNEL_DIR)/.config
	@$(MAKE) -C $(KERNEL_DIR) olddefconfig

menuconfig: extract
	@cp config/$(CONFIG).config $(KERNEL_DIR)/.config 2>/dev/null || true
	@$(MAKE) -C $(KERNEL_DIR) menuconfig
	@cp $(KERNEL_DIR)/.config config/$(CONFIG).config

build: configure
	@echo "==> Building kernel..."
	@$(MAKE) -C $(KERNEL_DIR) -j$(NPROC) bzImage
	@$(MAKE) -C $(KERNEL_DIR) -j$(NPROC) modules
	@mkdir -p $(BUILD_DIR)
	@cp $(KERNEL_DIR)/arch/x86/boot/bzImage $(BUILD_DIR)/vmlinuz-$(KERNEL_VERSION)-mixos
	@echo "==> Kernel built: $(BUILD_DIR)/vmlinuz-$(KERNEL_VERSION)-mixos"

modules_install: build
	@echo "==> Installing modules..."
	@mkdir -p $(MODULES_DIR)
	@$(MAKE) -C $(KERNEL_DIR) INSTALL_MOD_PATH=$(MODULES_DIR) modules_install
	@echo "==> Modules installed to $(MODULES_DIR)"

clean:
	@echo "==> Cleaning..."
	@rm -rf $(KERNEL_DIR) $(BUILD_DIR)
	@rm -f linux-$(KERNEL_VERSION).tar.xz

distclean: clean
	@rm -rf ../build/kernel
