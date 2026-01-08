# MIXOS GO Developer Profile
# Full development environment

[profile]
name = developer
description = Complete development environment with containers and tools
size_estimate = 8GB

[packages]
# Include base
@base

# Development Tools
git
git-lfs
make
cmake
ninja
gcc
g++
clang
llvm
gdb
valgrind
strace
ltrace

# Languages
python
python-pip
python-virtualenv
nodejs
npm
yarn
go
rust
cargo

# Editors
vim
neovim
nano
emacs

# Shell Tools
tmux
screen
htop
btop
ncdu
tree
ripgrep
fd
fzf
jq
yq

# Containers
docker
docker-compose
podman
buildah
skopeo
kubectl
helm

# Networking
curl
wget
httpie
nmap
tcpdump
wireshark-cli
socat
netcat

# Database Clients
postgresql-client
mysql-client
redis-cli
mongodb-tools

# Cloud Tools
aws-cli
azure-cli
gcloud

# Version Control
git
mercurial
subversion

# Documentation
man-db
man-pages
tldr

[services]
@base
docker.service
containerd.service

[config]
default_shell = /bin/zsh
enable_ssh = true
enable_docker = true
enable_gui = false
setup_dotfiles = true
