# MIXOS GO ISO Build Makefile

VERSION ?= 1.0
KERNEL_VERSION ?= 6.6.10-mixos
ARCH ?= x86_64

ISO_DIR := $(shell pwd)
ROOT_DIR := $(shell dirname $(ISO_DIR))
BUILD_DIR := $(ROOT_DIR)/build/iso
WORK_DIR := $(BUILD_DIR)/work
OUTPUT := $(BUILD_DIR)/mixos-go-$(VERSION)-$(ARCH).iso

.PHONY: all clean iso efi-image prepare

all: iso

clean:
	rm -rf $(BUILD_DIR)

prepare:
	mkdir -p $(WORK_DIR)/{boot/grub,EFI/BOOT,LiveOS,isolinux}

efi-image: prepare
	@echo "Creating EFI boot image..."
	dd if=/dev/zero of=$(WORK_DIR)/efi.img bs=1M count=64
	mkfs.vfat -F 32 $(WORK_DIR)/efi.img
	mmd -i $(WORK_DIR)/efi.img ::/EFI ::/EFI/BOOT
	mcopy -i $(WORK_DIR)/efi.img $(ISO_DIR)/efi/EFI/BOOT/grub.cfg ::/EFI/BOOT/
	@if [ -f $(WORK_DIR)/EFI/BOOT/BOOTX64.EFI ]; then \
		mcopy -i $(WORK_DIR)/efi.img $(WORK_DIR)/EFI/BOOT/BOOTX64.EFI ::/EFI/BOOT/; \
	fi

iso: prepare
	@echo "Building MIXOS GO ISO..."
	bash $(ISO_DIR)/scripts/build-iso.sh

test:
	@echo "Testing ISO with QEMU..."
	qemu-system-x86_64 \
		-cdrom $(OUTPUT) \
		-m 2G \
		-enable-kvm \
		-boot d

test-uefi:
	@echo "Testing ISO with QEMU (UEFI)..."
	qemu-system-x86_64 \
		-cdrom $(OUTPUT) \
		-m 2G \
		-enable-kvm \
		-bios /usr/share/ovmf/OVMF.fd \
		-boot d
