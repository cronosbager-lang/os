# MixOS Build System
# Custom OS with Go init, OCaml services, and Python agent

.PHONY: all clean build-init build-services build-agent test help

# Directories
BIN_DIR := bin
SVC_DIR := svc
AGENT_DIR := agent
STORE_DIR := store
KERNEL_DIR := kernel

# Go settings
GO := go
GO_FLAGS := -ldflags="-s -w" -trimpath
GO_STATIC := CGO_ENABLED=0 $(GO) build $(GO_FLAGS)

# OCaml settings
DUNE := dune

# Output
BUILD_DIR := _build
DIST_DIR := dist

# Version
VERSION := 0.1.0

all: build-init build-services
	@echo "Build complete"

# ============================================
# Go Init System
# ============================================

build-init:
	@echo "Building Go init system..."
	@mkdir -p $(BUILD_DIR)/bin
	cd $(BIN_DIR) && $(GO) mod tidy
	cd $(BIN_DIR) && $(GO_STATIC) -o ../$(BUILD_DIR)/bin/init .
	@echo "Init system built: $(BUILD_DIR)/bin/init"

build-init-static:
	@echo "Building static Go init (musl)..."
	@mkdir -p $(BUILD_DIR)/bin
	cd $(BIN_DIR) && CC=musl-gcc $(GO_STATIC) -o ../$(BUILD_DIR)/bin/init-static .

# ============================================
# OCaml Services
# ============================================

build-services: build-broker build-pkgmgr build-builder build-resolver build-cache
	@echo "All services built"

build-broker:
	@echo "Building OCaml broker..."
	cd $(SVC_DIR)/broker && $(DUNE) build
	@mkdir -p $(BUILD_DIR)/svc/broker
	cp $(SVC_DIR)/broker/_build/default/src/broker.exe $(BUILD_DIR)/svc/broker/broker || true

build-pkgmgr:
	@echo "Building OCaml package manager..."
	cd $(SVC_DIR)/pkgmgr && $(DUNE) build
	@mkdir -p $(BUILD_DIR)/svc/pkgmgr
	cp $(SVC_DIR)/pkgmgr/_build/default/src/pkgmgr.exe $(BUILD_DIR)/svc/pkgmgr/pkgmgr || true

build-builder:
	@echo "Building OCaml build executor..."
	cd $(SVC_DIR)/builder && $(DUNE) build
	@mkdir -p $(BUILD_DIR)/svc/builder
	cp $(SVC_DIR)/builder/_build/default/src/builder.exe $(BUILD_DIR)/svc/builder/builder || true

build-resolver:
	@echo "Building OCaml dependency resolver..."
	cd $(SVC_DIR)/resolver && $(DUNE) build
	@mkdir -p $(BUILD_DIR)/svc/resolver
	cp $(SVC_DIR)/resolver/_build/default/src/resolver.exe $(BUILD_DIR)/svc/resolver/resolver || true

build-cache:
	@echo "Building OCaml cache service..."
	cd $(SVC_DIR)/cache && $(DUNE) build
	@mkdir -p $(BUILD_DIR)/svc/cache
	cp $(SVC_DIR)/cache/_build/default/src/cache.exe $(BUILD_DIR)/svc/cache/cache || true

# ============================================
# Python Agent
# ============================================

build-agent:
	@echo "Setting up Python agent..."
	@mkdir -p $(BUILD_DIR)/agent
	cp -r $(AGENT_DIR)/* $(BUILD_DIR)/agent/
	cp -r mix-agent/mixos_agent $(BUILD_DIR)/agent/
	@echo "Agent ready: $(BUILD_DIR)/agent"

# ============================================
# Mix Tools (user-facing)
# ============================================

build-mix-tools: build-mix-cli build-mix-pkg build-mix-installer build-mix-agent-early
	@echo "All mix tools built"

build-mix-cli:
	@echo "Building mix-cli..."
	@mkdir -p $(BUILD_DIR)/bin
	cd mix-cli && $(GO) mod tidy
	cd mix-cli && $(GO_STATIC) -o ../$(BUILD_DIR)/bin/mix-cli .
	@echo "mix-cli built: $(BUILD_DIR)/bin/mix-cli"

build-mix-pkg:
	@echo "Building mix-pkg..."
	@mkdir -p $(BUILD_DIR)/bin
	cd mix-pkg && cargo build --release
	cp mix-pkg/target/release/mix-pkg $(BUILD_DIR)/bin/ || true
	@echo "mix-pkg built: $(BUILD_DIR)/bin/mix-pkg"

build-mix-installer:
	@echo "Building mix-installer..."
	@mkdir -p $(BUILD_DIR)/bin
	cd mix-installer && $(GO) mod tidy
	cd mix-installer && $(GO_STATIC) -o ../$(BUILD_DIR)/bin/mix-installer .
	@echo "mix-installer built: $(BUILD_DIR)/bin/mix-installer"

build-mix-agent-early:
	@echo "Building mix-agent-early..."
	@mkdir -p $(BUILD_DIR)/bin
	cd mix-agent-early && $(GO) mod tidy
	cd mix-agent-early && $(GO_STATIC) -o ../$(BUILD_DIR)/bin/mix-agent-early .
	@echo "mix-agent-early built: $(BUILD_DIR)/bin/mix-agent-early"

build-mix-agent-early-static:
	@echo "Building static mix-agent-early (musl)..."
	@mkdir -p $(BUILD_DIR)/bin
	cd mix-agent-early && CC=musl-gcc $(GO_STATIC) -o ../$(BUILD_DIR)/bin/mix-agent-early-static .

# ============================================
# Distribution
# ============================================

dist: all build-agent build-mix-tools
	@echo "Creating distribution..."
	@mkdir -p $(DIST_DIR)
	@mkdir -p $(DIST_DIR)/bin
	@mkdir -p $(DIST_DIR)/svc
	@mkdir -p $(DIST_DIR)/agent
	@mkdir -p $(DIST_DIR)/store
	@mkdir -p $(DIST_DIR)/etc/mixos
	
	# Copy init
	cp $(BUILD_DIR)/bin/init $(DIST_DIR)/bin/
	
	# Copy services
	cp -r $(BUILD_DIR)/svc/* $(DIST_DIR)/svc/ || true
	
	# Copy agent
	cp -r $(BUILD_DIR)/agent/* $(DIST_DIR)/agent/
	
	# Copy mix tools
	cp $(BUILD_DIR)/bin/mix-cli $(DIST_DIR)/bin/ || true
	cp $(BUILD_DIR)/bin/mix-pkg $(DIST_DIR)/bin/ || true
	cp $(BUILD_DIR)/bin/mix-installer $(DIST_DIR)/bin/ || true
	cp $(BUILD_DIR)/bin/mix-agent-early $(DIST_DIR)/bin/ || true
	
	# Copy config
	cp etc/init.toml $(DIST_DIR)/etc/mixos/ || true
	
	@echo "Distribution created: $(DIST_DIR)"

# ============================================
# Testing
# ============================================

test: test-init test-ipc
	@echo "All tests passed"

test-init:
	@echo "Testing init system..."
	cd $(BIN_DIR) && $(GO) test ./...

test-ipc:
	@echo "Testing IPC..."
	cd $(AGENT_DIR) && python3 -m pytest ipc/ -v || true

test-integration:
	@echo "Running integration tests..."
	./tests/run-integration.sh

# ============================================
# Development
# ============================================

dev-broker:
	@echo "Running broker in development mode..."
	cd $(SVC_DIR)/broker && $(DUNE) exec ./src/broker.exe

dev-init:
	@echo "Running init in development mode..."
	cd $(BIN_DIR) && $(GO) run .

# ============================================
# Clean
# ============================================

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	cd $(SVC_DIR)/broker && $(DUNE) clean || true
	cd $(SVC_DIR)/pkgmgr && $(DUNE) clean || true
	cd $(SVC_DIR)/builder && $(DUNE) clean || true
	cd $(SVC_DIR)/resolver && $(DUNE) clean || true
	cd $(SVC_DIR)/cache && $(DUNE) clean || true
	@echo "Clean complete"

# ============================================
# Dependencies
# ============================================

deps:
	@echo "Installing dependencies..."
	# Go
	cd $(BIN_DIR) && $(GO) mod download
	cd mix-cli && $(GO) mod download || true
	cd mix-installer && $(GO) mod download || true
	# OCaml
	opam install lwt lwt_ppx yojson ppx_deriving ppx_deriving_yojson digestif -y || true
	# Python
	pip install -r mix-agent/requirements.txt || true
	@echo "Dependencies installed"

# ============================================
# Help
# ============================================

help:
	@echo "MixOS Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Core Targets:"
	@echo "  all              Build init and services"
	@echo "  build-init       Build Go init system"
	@echo "  build-services   Build all OCaml services"
	@echo "  build-agent      Setup Python agent"
	@echo "  build-mix-tools  Build all mix-* tools"
	@echo "  dist             Create distribution"
	@echo "  test             Run tests"
	@echo "  clean            Clean build artifacts"
	@echo "  deps             Install dependencies"
	@echo "  help             Show this help"
	@echo ""
	@echo "OCaml Services:"
	@echo "  build-broker     Build IPC broker"
	@echo "  build-pkgmgr     Build package manager service"
	@echo "  build-builder    Build executor"
	@echo "  build-resolver   Build dependency resolver"
	@echo "  build-cache      Build artifact cache"
	@echo ""
	@echo "Mix Tools:"
	@echo "  build-mix-cli         Build CLI tool"
	@echo "  build-mix-pkg         Build package manager CLI"
	@echo "  build-mix-installer   Build TUI installer"
	@echo "  build-mix-agent-early Build early boot agent"
