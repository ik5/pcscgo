#!/usr/bin/env bash
set -euo pipefail

# Test script for pcscgo - builds and runs tests for all three binding modes
# Usage: ./test_cgo.sh [mode]
#   mode: purego, dynamic, static, or all (default)

MODE="${1:-all}"

echo "=== Testing pcscgo (mode: $MODE) ==="

run_purego() {
    echo ""
    echo "--- Purego mode (no CGO) ---"
    
    # amd64
    echo "Building amd64..."
    CGO_ENABLED=0 go build -tags=pcsc_purego ./...
    echo "Testing amd64..."
    CGO_ENABLED=0 go test -tags=pcsc_purego -v ./...
    
    # 386 (cross-compile build only, tests run on host)
    echo "Building 386..."
    CGO_ENABLED=0 GOARCH=386 go build -tags=pcsc_purego ./...
    echo "Testing 386..."
    CGO_ENABLED=0 GOARCH=386 go test -tags=pcsc_purego -v ./...
}

run_dynamic() {
    echo ""
    echo "--- Dynamic linking (CGO) ---"
    
    # amd64
    echo "Building amd64..."
    CGO_ENABLED=1 CC="gcc -m64" PKG_CONFIG_PATH=/usr/lib/pkgconfig go build -tags=pcsc_dynamic ./...
    echo "Testing amd64..."
    CGO_ENABLED=1 go test -tags=pcsc_dynamic -v ./...
    
    # 386 - requires 32-bit toolchain and libpcsclite
    if command -v gcc >/dev/null 2>&1 && gcc -m32 -v 2>&1 | grep -q "Target: i686"; then
        echo "Building 386..."
        CGO_ENABLED=1 GOARCH=386 CC="gcc -m32" PKG_CONFIG_PATH=/usr/lib32/pkgconfig go build -tags=pcsc_dynamic ./...
        echo "Testing 386..."
        CGO_ENABLED=1 GOARCH=386 go test -tags=pcsc_dynamic -v ./...
    else
        echo "Skipping 386 (32-bit toolchain not available)"
    fi
}

run_static() {
    echo ""
    echo "--- Static linking (CGO) ---"
    
    # amd64
    echo "Building amd64..."
    CGO_ENABLED=1 CC="gcc -m64" PKG_CONFIG_PATH=/usr/lib/pkgconfig go build -tags=pcsc_static ./...
    echo "Testing amd64..."
    CGO_ENABLED=1 go test -tags=pcsc_static -v ./...
    
    # 386 - requires 32-bit toolchain and static libpcsclite
    if command -v gcc >/dev/null 2>&1 && gcc -m32 -v 2>&1 | grep -q "Target: i686"; then
        echo "Building 386..."
        CGO_ENABLED=1 GOARCH=386 CC="gcc -m32" PKG_CONFIG_PATH=/usr/lib32/pkgconfig go build -tags=pcsc_static ./...
        echo "Testing 386..."
        CGO_ENABLED=1 GOARCH=386 go test -tags=pcsc_static -v ./...
    else
        echo "Skipping 386 (32-bit toolchain not available)"
    fi
}

run_vet() {
    echo ""
    echo "--- go vet ---"
    CGO_ENABLED=0 go vet -tags=pcsc_purego ./...
    CGO_ENABLED=1 go vet -tags=pcsc_dynamic ./...
    CGO_ENABLED=1 go vet -tags=pcsc_static ./...
}

run_fmt() {
    echo ""
    echo "--- go fmt ---"
    go fmt ./...
}

case "$MODE" in
    purego)
        run_purego
        ;;
    dynamic)
        run_dynamic
        ;;
    static)
        run_static
        ;;
    all)
        run_purego
        run_dynamic
        run_static
        ;;
    vet)
        run_vet
        ;;
    fmt)
        run_fmt
        ;;
    *)
        echo "Usage: $0 [purego|dynamic|static|all|vet|fmt]"
        exit 1
        ;;
esac

echo ""
echo "=== All tests passed for mode: $MODE ==="