# Makefile for PC/SC Smart Card Scanner Example
#
# This Makefile builds the scanner in three different linking modes:
#   - scan     : purego (runtime library loading)
#   - dyn_scan : dynamic CGO linking
#   - st_scan  : static CGO linking
#
# The purego binary accepts a -lib flag to specify the library path.
# The CGO binaries link against libpcsclite at compile time.
#
# Usage:
#   make              # Build all three binaries
#   make scan         # Build purego version only
#   make dyn_scan     # Build dynamic CGO version only
#   make st_scan      # Build static CGO version only
#   make test         # Run tests for all modes
#   make clean        # Remove all built binaries

# Go configuration
GO        := go
GOFLAGS   := -trimpath
# Project root directory (where go.work lives)
PROJECT_ROOT := $(CURDIR)/../..
GOWORK    := $(PROJECT_ROOT)/go.work

# Build tags for each mode
PUREGO_TAGS   := pcsc_purego
DYNAMIC_TAGS  := pcsc_dynamic
STATIC_TAGS   := pcsc_static

# CGO settings
CGO_ENABLED_DYNAMIC := CGO_ENABLED=1
CGO_ENABLED_STATIC  := CGO_ENABLED=1

# Library paths (for pkg-config)
PKG_CONFIG_PATH_DYNAMIC := /usr/lib/pkgconfig
PKG_CONFIG_PATH_STATIC  := /usr/lib/pkgconfig

# Output binary names
SCAN_BIN     := scan
DYN_SCAN_BIN := dyn_scan
ST_SCAN_BIN  := st_scan

# Source directory
SRC_DIR := .

# Default target
.PHONY: all
all: $(SCAN_BIN) $(DYN_SCAN_BIN) $(ST_SCAN_BIN)

# ============================================================
# PUREGO MODE (runtime library loading)
# ============================================================
# The purego version uses github.com/ebitengine/purego to load
# libpcsclite.so at runtime via dlopen/dlsym.
# No CGO required, cross-compiles easily.
# Run with: ./scan [-lib /path/to/libpcsclite.so]

$(SCAN_BIN): $(SRC_DIR)/main.go
	@echo "Building purego version -> $@"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) CGO_ENABLED=0 $(GO) build $(GOFLAGS) -tags=$(PUREGO_TAGS) -o examples/scan/$@ ./examples/scan
	@echo "  Built: $@ (purego)"

# ============================================================
# DYNAMIC CGO MODE (runtime dynamic linking)
# ============================================================
# Uses CGO with #cgo pkg-config: libpcsclite
# Links against shared libpcsclite.so at runtime.
# Requires CGO toolchain and libpcsclite development headers.

$(DYN_SCAN_BIN): $(SRC_DIR)/main.go
	@echo "Building dynamic CGO version -> $@"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) $(CGO_ENABLED_DYNAMIC) PKG_CONFIG_PATH=$(PKG_CONFIG_PATH_DYNAMIC) \
		$(GO) build $(GOFLAGS) -tags=$(DYNAMIC_TAGS) -o examples/scan/$@ ./examples/scan
	@echo "  Built: $@ (dynamic CGO)"

# ============================================================
# STATIC CGO MODE (compile-time static linking)
# ============================================================
# Uses CGO with #cgo pkg-config: --static libpcsclite
# Links libpcsclite.a and its dependencies statically.
# Produces a standalone binary with no runtime library dependency.
# Requires static libpcsclite.a and all its dependencies (libusb, etc.)

$(ST_SCAN_BIN): $(SRC_DIR)/main.go
	@echo "Building static CGO version -> $@"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) $(CGO_ENABLED_STATIC) PKG_CONFIG_PATH=$(PKG_CONFIG_PATH_STATIC) \
		$(GO) build $(GOFLAGS) -tags=$(STATIC_TAGS) -o examples/scan/$@ ./examples/scan
	@echo "  Built: $@ (static CGO)"

# ============================================================
# 32-bit builds (if toolchain available)
# ============================================================
# These targets build 386 versions. They require a 32-bit toolchain
# and 32-bit libpcsclite development packages.

.PHONY: scan386
scan386: $(SRC_DIR)/main.go
	@echo "Building purego 386 version -> $(SCAN_BIN)386"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) CGO_ENABLED=0 GOARCH=386 $(GO) build $(GOFLAGS) -tags=$(PUREGO_TAGS) -o examples/scan/$(SCAN_BIN)386 ./examples/scan
	@echo "  Built: $(SCAN_BIN)386 (purego 386)"

.PHONY: dyn_scan386
dyn_scan386: $(SRC_DIR)/main.go
	@echo "Building dynamic CGO 386 version -> $(DYN_SCAN_BIN)386"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) $(CGO_ENABLED_DYNAMIC) GOARCH=386 CC="gcc -m32" PKG_CONFIG_PATH=/usr/lib32/pkgconfig \
		$(GO) build $(GOFLAGS) -tags=$(DYNAMIC_TAGS) -o examples/scan/$(DYN_SCAN_BIN)386 ./examples/scan
	@echo "  Built: $(DYN_SCAN_BIN)386 (dynamic CGO 386)"

.PHONY: st_scan386
st_scan386: $(SRC_DIR)/main.go
	@echo "Building static CGO 386 version -> $(ST_SCAN_BIN)386"
	@cd $(PROJECT_ROOT) && \
		GOWORK=$(GOWORK) $(CGO_ENABLED_STATIC) GOARCH=386 CC="gcc -m32" PKG_CONFIG_PATH=/usr/lib32/pkgconfig \
		$(GO) build $(GOFLAGS) -tags=$(STATIC_TAGS) -o examples/scan/$(ST_SCAN_BIN)386 ./examples/scan
	@echo "  Built: $(ST_SCAN_BIN)386 (static CGO 386)"

# ============================================================
# Testing
# ============================================================

.PHONY: test
test: test-purego test-dynamic test-static

.PHONY: test-purego
test-purego:
	@echo "Running purego tests..."
	@cd $(PROJECT_ROOT) && GOWORK=$(GOWORK) CGO_ENABLED=0 $(GO) test -tags=$(PUREGO_TAGS) -v ./...

.PHONY: test-dynamic
test-dynamic:
	@echo "Running dynamic CGO tests..."
	@cd $(PROJECT_ROOT) && GOWORK=$(GOWORK) $(CGO_ENABLED_DYNAMIC) PKG_CONFIG_PATH=$(PKG_CONFIG_PATH_DYNAMIC) $(GO) test -tags=$(DYNAMIC_TAGS) -v ./...

.PHONY: test-static
test-static:
	@echo "Running static CGO tests..."
	@cd $(PROJECT_ROOT) && GOWORK=$(GOWORK) $(CGO_ENABLED_STATIC) PKG_CONFIG_PATH=$(PKG_CONFIG_PATH_STATIC) $(GO) test -tags=$(STATIC_TAGS) -v ./...

# ============================================================
# Development helpers
# ============================================================

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	@CGO_ENABLED=0 $(GO) vet -tags=$(PUREGO_TAGS) ./...
	@$(CGO_ENABLED_DYNAMIC) $(GO) vet -tags=$(DYNAMIC_TAGS) ./...
	@$(CGO_ENABLED_STATIC) $(GO) vet -tags=$(STATIC_TAGS) ./...

.PHONY: lint
lint: fmt vet

# ============================================================
# Cleanup
# ============================================================

.PHONY: clean
clean:
	@echo "Cleaning binaries..."
	@rm -f $(SCAN_BIN) $(DYN_SCAN_BIN) $(ST_SCAN_BIN)
	@rm -f $(SCAN_BIN)386 $(DYN_SCAN_BIN)386 $(ST_SCAN_BIN)386
	@cd $(PROJECT_ROOT) && rm -f examples/scan/$(SCAN_BIN) examples/scan/$(DYN_SCAN_BIN) examples/scan/$(ST_SCAN_BIN)
	@cd $(PROJECT_ROOT) && rm -f examples/scan/$(SCAN_BIN)386 examples/scan/$(DYN_SCAN_BIN)386 examples/scan/$(ST_SCAN_BIN)386
	@echo "Done"

# ============================================================
# Run helpers
# ============================================================

.PHONY: run-scan
run-scan: $(SCAN_BIN)
	@echo "Running purego scanner..."
	@./$(SCAN_BIN) -v

.PHONY: run-dyn
run-dyn: $(DYN_SCAN_BIN)
	@echo "Running dynamic scanner..."
	@./$(DYN_SCAN_BIN) -v

.PHONY: run-st
run-st: $(ST_SCAN_BIN)
	@echo "Running static scanner..."
	@./$(ST_SCAN_BIN) -v

# ============================================================
# Help
# ============================================================

.PHONY: help
help:
	@echo "PC/SC Scanner Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build all three binaries (default)"
	@echo "  scan         Build purego version (runtime loading)"
	@echo "  dyn_scan     Build dynamic CGO version"
	@echo "  st_scan      Build static CGO version"
	@echo "  scan386      Build purego for 386"
	@echo "  dyn_scan386  Build dynamic CGO for 386"
	@echo "  st_scan386   Build static CGO for 386"
	@echo ""
	@echo "  test         Run tests for all modes"
	@echo "  test-purego  Run purego tests"
	@echo "  test-dynamic Run dynamic CGO tests"
	@echo "  test-static  Run static CGO tests"
	@echo ""
	@echo "  fmt          Format source code"
	@echo "  vet          Run go vet on all modes"
	@echo "  lint         Run fmt and vet"
	@echo ""
	@echo "  clean        Remove all built binaries"
	@echo ""
	@echo "  run-scan     Build and run purego scanner"
	@echo "  run-dyn      Build and run dynamic scanner"
	@echo "  run-st       Build and run static scanner"
	@echo ""
	@echo "Environment variables:"
	@echo "  CGO_ENABLED_DYNAMIC  (default: 1)"
	@echo "  CGO_ENABLED_STATIC   (default: 1)"
	@echo "  PKG_CONFIG_PATH_DYNAMIC (default: /usr/lib/pkgconfig)"
	@echo "  PKG_CONFIG_PATH_STATIC  (default: /usr/lib/pkgconfig)"