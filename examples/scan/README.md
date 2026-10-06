# PC/SC Smart Card Scanner Example

A complete implementation of a smart card reader monitor similar to the standard `pcsc_scan` utility, demonstrating proper usage of the [pcscgo](https://github.com/ik5/pcscgo) library.

## Features

- **Reader monitoring**: Detects card insertion/removal
- **Reader hot-plug**: Detects reader connect/disconnect via PnP notifications
- **Signal handling**: Proper Ctrl+C/SIGTERM handling using `SCardCancel` (like `pcsc_scan`)
- **Three build modes**: purego, dynamic CGO, static CGO
- **Memory safe**: Proper memory pinning for CGO backends
- **Cross-platform**: Linux, FreeBSD, OpenBSD, NetBSD

## Quick Start

```bash
# Build all three versions
make all

# Run purego version (runtime library loading)
./scan -v

# Run dynamic CGO version
./dyn_scan -v

# Run static CGO version (no runtime dependencies)
./st_scan -v
```

## Build Modes

| Mode | Command | Description |
|------|---------|-------------|
| **purego** | `make scan` | Runtime loading via `dlopen`/`dlsym` using [purego](https://github.com/ebitengine/purego). No CGO required. Cross-compiles easily. |
| **dynamic** | `make dyn_scan` | CGO with dynamic linking to `libpcsclite.so`. Standard CGO approach. |
| **static** | `make st_scan` | CGO with static linking (`libpcsclite.a` + dependencies). No runtime library dependency. |

### Building

```bash
# Build all three binaries
make all

# Build specific mode
make scan       # purego
make dyn_scan   # dynamic CGO
make st_scan    # static CGO

# Cross-compile for 386 (requires 32-bit toolchain)
make scan386
make dyn_scan386
make st_scan386

# Clean
make clean
```

### Testing

```bash
# Run tests for all modes
make test

# Specific mode tests
make test-purego
make test-dynamic
make test-static
```

## Running

```bash
# Default: INFINITE timeout (event-driven, like pcsc_scan)
./scan -v

# Custom timeout (polling mode, useful for debugging)
./scan -timeout 500ms -v

# Polling mode (1 second interval)
./scan -timeout 0 -v

# Purego with custom library path
./scan -lib /usr/lib/x86_64-linux-gnu/libpcsclite.so.1 -v

# Help
./scan -h
```

### Timeout Behavior

| Flag | Behavior |
|------|----------|
| `-timeout -1` (default) | **INFINITE** - blocks until state change (like `pcsc_scan`) |
| `-timeout 0` | 1000ms polling interval |
| `-timeout 500ms` | Custom polling interval |
| `-timeout -1ms` | INFINITE (explicit) |

**Recommendation**: Use the default INFINITE timeout for production. Polling modes are for debugging only.

## Signal Handling

The scanner implements proper signal handling like `pcsc_scan`:

| Signal | Behavior |
|--------|----------|
| `Ctrl+C` (SIGINT) | Graceful shutdown - interrupts blocking call via `SCardCancel` |
| `Ctrl+\` (SIGQUIT) | Force exit after 500ms grace period |
| `SIGTERM` | Graceful shutdown |

The key technique: `SCardCancel` is called from the signal handler to interrupt the blocking `SCardGetStatusChange` call, causing it to return `SCardErrorCancelled` which is handled as graceful shutdown.

## Output Examples

### Card Events

```
[ACS ACR39U ICC Reader 01 00] CARD INSERTED
  ATR: 3B 8F 80 01 80 4F 0C A0 00 00 03 06 03 00 01 00 00 00 00 6A
[ACS ACR39U ICC Reader 01 00] CARD REMOVED
```

### Reader Events

```
[ACS ACR39U ICC Reader 01 00] READER DISCONNECTED (unavailable)
READER REMOVED: (reader disconnected via PnP)
...
[New Reader Name] READER CONNECTED (available)
```

### Verbose Output

```
[Reader Name] State changed: 0x12 -> 0x22
  State flags: Present | ATRMatch
```

## Architecture

```
scanner/
├── main.go       # Main implementation
├── doc.go        # Package documentation (godoc)
├── load.go       # Purego library loader (build tag: pcsc_purego)
├── load_cgo.go   # CGO no-op loader (build tag: !pcsc_purego)
├── go.mod        # Module with replace directive for local pcscgo
├── Makefile      # Build system for all three modes
└── README.md     # This file
```

### Key Components

1. **Scanner struct** - Encapsulates all state (context, readers, states, status map)
2. **Signal handling** - Uses `SCardCancel` to interrupt blocking calls (like `pcsc_scan`)
3. **Memory pinning** - `runtime.Pinner` prevents GC moves during CGO calls
4. **State tracking** - Per-reader status map detects transitions
5. **PnP handling** - Re-enumerates on `\\?PnP?\Notification` events

### State Machine

```
Initial (Unaware)
    │
    ▼ (first SCardGetStatusChange returns current state)
Current State ───▶ Event State (on change)
    │                    │
    ▼                    ▼
Update Current = Event  Process transition
    │                    │
    └────────────────────┘ (loop)
```

## Requirements

- Go 1.21+
- pcscd running (`systemctl start pcscd`)
- libpcsclite development headers (for CGO builds):
  - Debian/Ubuntu: `libpcsclite-dev`
  - Arch: `pcsc-lite`
  - Fedora: `pcsc-lite-devel`
  - FreeBSD: `devel/pcsc-lite`

### Purego Mode Only

- No CGO toolchain required
- libpcsclite.so must be in library path or `PCSCLITE_LIB_PATH` set

## Troubleshooting

### Reader Not Detected

1. Check pcscd is running: `systemctl status pcscd`
2. Check USB device: `lsusb | grep -i <reader>`
3. Check pcscd logs: `journalctl -u pcscd -f`
3. Verify udev rules allow access

### ACS ACR39U Not Showing

The ACR39U uses the CCID driver. Ensure:
- `pcscd` sees the USB device (check `journalctl -u pcscd`)
- No other process has exclusive access
- Try unplugging/replugging the reader

### Permission Errors

Add user to `pcscd` group or configure polkit:
```bash
sudo usermod -a -G pcscd $USER
# or create /etc/polkit-1/localauthority/50-local.d/pcscd.pkla
```

### Build Errors

- **CGO**: Ensure gcc and pkg-config are installed
- **Static**: Requires static libraries (libpcsclite.a, libusb.a, etc.)
- **Purego**: No build dependencies beyond Go

## Godoc

```bash
# View package documentation
go doc -tags=pcsc_purego github.com/ik5/pcscgo/examples/scan

# View specific function
go doc -tags=pcsc_purego github.com/ik5/pcscgo/examples/scan Scanner.Run
```

## License

Same as pcscgo - see root LICENSE file.

## References

- [PC/SC Workgroup Specifications](https://pcscworkgroup.com/)
- [pcsc-lite Documentation](https://pcsclite.apdu.fr/)
- [pcsc_scan Source](https://github.com/LudovicRousseau/pcsc-lite/blob/master/src/tools/pcsc_scan.c)
- [purego Library](https://github.com/ebitengine/purego)