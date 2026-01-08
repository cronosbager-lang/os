# MIXOS GO Full Profile
# Complete desktop environment with all features

[profile]
name = full
description = Full desktop installation with GUI and all features
size_estimate = 15GB

[packages]
# Include developer profile
@developer

# Desktop Environment
xorg-server
xorg-xinit
xorg-apps
mesa
mesa-utils

# Window Manager / Desktop
sway
waybar
wofi
mako
foot
alacritty

# Or GNOME
# gnome-shell
# gnome-terminal
# nautilus
# gnome-control-center

# Or KDE
# plasma-desktop
# konsole
# dolphin
# systemsettings

# Audio
pipewire
pipewire-pulse
pipewire-alsa
wireplumber
pavucontrol

# Video
ffmpeg
mpv
vlc

# Graphics
gimp
inkscape
imagemagick

# Office
libreoffice

# Browsers
firefox
chromium

# Communication
discord
slack
telegram-desktop

# Fonts
ttf-dejavu
ttf-liberation
noto-fonts
noto-fonts-emoji
ttf-fira-code
ttf-jetbrains-mono

# Themes
papirus-icon-theme
arc-gtk-theme

# Utilities
gparted
baobab
gnome-disk-utility
file-roller

# Printing
cups
cups-pdf
system-config-printer

# Bluetooth
bluez
bluez-utils
blueman

[services]
@developer
cups.service
bluetooth.service

[config]
default_shell = /bin/zsh
enable_ssh = true
enable_docker = true
enable_gui = true
default_session = sway
enable_bluetooth = true
enable_printing = true
