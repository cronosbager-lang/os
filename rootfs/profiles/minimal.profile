# MIXOS GO Minimal Profile
# Bare minimum for a bootable system

[profile]
name = minimal
description = Minimal installation with essential packages only
size_estimate = 2GB

[packages]
# Core only
base
linux
linux-firmware
systemd
e2fsprogs
grub
networkmanager
openssh
sudo
bash
coreutils
util-linux
mix-cli
mix-pkg
mix-agent

[services]
systemd-journald.service
systemd-udevd.service
NetworkManager.service
sshd.service
mixos-agent.service

[config]
# Minimal config
default_shell = /bin/bash
enable_ssh = true
enable_docker = false
enable_gui = false
