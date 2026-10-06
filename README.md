# pcscgo

[![Go Reference](https://pkg.go.dev/badge/github.com/ik5/pcscgo.svg)](https://pkg.go.dev/github.com/ik5/pcscgo)
[![License](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)

**pcscgo** is a pure Go binding for PC/SC (PC/SC-Lite) smart card middleware. It enables Go applications to communicate with smart card readers via the PC/SC standard.

## Features

- **Three linking modes** — choose at compile time:
  - `pcsc_purego` — Runtime loading via `dlopen`/`dlsym` (no CGO required)
  - `pcsc_dynamic` — CGO with dynamic linking to `libpcsclite.so`
  - `pcsc_static` — CGO with static linking (`libpcsclite.a` + dependencies)
- **Modern Go API** — Returns `error` instead of raw error codes
- **Proper error types** — `PCSCError`, `LoadError`, `ErrorCode` with `errors.Is/As` support
- **Purego mode** — Runtime library loading via `dlopen`/`dlsym` (no CGO toolchain needed)
- **Signal-safe** — Uses `SCardCancel` for graceful Ctrl+C handling
- **Memory safe** — Proper `runtime.Pinner` usage for CGO backends
- **Hot-plug support** — PnP notifications for reader connect/disconnect

## Understanding PC/SC

Before using pcscgo, it helps to understand the PC/SC architecture and how this package fits in.

### What is PC/SC?

**PC/SC (Personal Computer/Smart Card)** is an industry standard for integrating smart cards into computing environments. It defines a standard API for applications to communicate with smart card readers and the cards themselves, abstracting away hardware differences.

The standard is maintained by the **PC/SC Workgroup** (comprising Microsoft, Apple, Gemalto, NXP, and others) and is implemented on Windows, Linux, macOS, and BSDs.

### How PC/SC Works

```
┌─────────────────┐     PC/SC API      ┌─────────────────┐
│   Your Go App   │ ◄─────────────────► │   pcscgo        │
└─────────────────┘                    └────────┬────────┘
                                                │
                                    ┌───────────▼───────────┐
                                    │   libpcsclite (C)     │
                                    │   (pcsc-lite lib)     │
                                    └───────────┬───────────┘
                                                │
                                    ┌───────────▼───────────┐
                                    │   pcscd (daemon)      │
                                    │   (resource manager)  │
                                    └───────────┬───────────┘
                                                │
                    ┌───────────────────────────┼───────────────────────────┐
                    ▼                           ▼                           ▼
            ┌───────────────┐           ┌───────────────┐           ┌───────────────┐
            │ Reader Driver │           │ Reader Driver │           │ Reader Driver │
            │   (CCID)      │           │   (CCID)      │           │   (Custom)    │
            └───────┬───────┘           └───────┬───────┘           └───────┬───────┘
                    │                           │                           │
                    ▼                           ▼                           ▼
            ┌───────────────┐           ┌───────────────┐           ┌───────────────┐
            │ Smart Card    │           │ Smart Card    │           │ Smart Card    │
            │ Reader        │           │ Reader        │           │ Reader        │
            └───────────────┘           └───────────────┘           └───────────────┘
```

**Key components:**

1. **pcscd (daemon)** — The PC/SC resource manager. It runs as a system service, manages reader connections, handles hot-plug events, and multiplexes access from multiple applications. **Must be running** for PC/SC to work.

2. **libpcsclite** — The C client library that applications link against. It communicates with pcscd via a Unix socket (or named pipe on Windows).

3. **Reader drivers (IFD handlers)** — Plugins that translate CCID/USB commands to reader-specific protocols. The most common is the CCID driver for USB smart card readers.

4. **Smart card readers** — Hardware devices (USB, serial, PCMCIA) that communicate with cards via ISO 7816 protocols.

### How pcscgo Fits In

pcscgo is a **Go binding** for the PC/SC API. It does not implement the PC/SC stack itself — it bridges Go code to the existing libpcsclite/pcscd infrastructure.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Your Go Application                       │
├─────────────────────────────────────────────────────────────────┤
│  pcscgo (this package)                                           │
│  ├─ Public API (winscard.go) — SCard* functions returning error │
│  ├─ Backend interface — abstracts the implementation             │
│  ├─ purego backend — runtime dlopen/dlsym (no CGO)              │
│  ├─ CGO dynamic backend — links libpcsclite.so                   │
│  └─ CGO static backend — links libpcsclite.a                     │
├─────────────────────────────────────────────────────────────────┤
│  libpcsclite.so / .a (C library)                                │
├─────────────────────────────────────────────────────────────────┤
│  pcscd (daemon) — manages readers, handles ISO 7816, APDU I/O  │
└─────────────────────────────────────────────────────────────────┘
```

**What pcscgo provides:**

- **Type-safe Go API** — Functions return `error` instead of raw `LONG` codes
- **Three linking modes** — Choose at compile time (purego/CGO dynamic/CGO static)
- **Memory safety** — Proper `runtime.Pinner` for CGO, no unsafe casts
- **Signal handling** — Uses `SCardCancel` for graceful interruption
- **Hot-plug support** — PnP notifications for reader connect/disconnect

**What pcscgo does NOT do:**

- ❌ Does not implement the PC/SC protocol stack
- ❌ Does not talk directly to readers (no USB/CCID implementation)
- ❌ Does not replace pcscd — pcscd must be running
- ❌ Does not parse card protocols (ISO 7816, EMV, etc.) — you send/receive raw APDUs

### Mental Model

Think of pcscgo as a **thin translation layer**:

1. **Context** = Your session with pcscd (`SCardEstablishContext`/`ReleaseContext`)
2. **Reader** = A named endpoint managed by pcscd (`SCardListReaders`, `SCardConnect`)
3. **Card handle** = An active session with a specific card in a reader
4. **APDU I/O** = Raw command/response pairs (`SCardTransmit`)

The protocol flow:
```
SCardEstablishContext → SCardListReaders → SCardConnect → 
SCardTransmit (APDU) → ... → SCardDisconnect → SCardReleaseContext
```

Monitoring for events uses a separate pattern:
```
SCardGetStatusChange (blocks until state change)
```

This is the same mental model as the C API — pcscgo just makes it idiomatic Go.

## Installation

```go
package main

import (
	"errors"
	"log"
	"strings"

	"github.com/ik5/pcscgo"
)

func main() {
	// 1. Establish context with PC/SC resource manager
	var ctx pcscgo.SCardContext
	err := pcscgo.SCardEstablishContext(pcscgo.SCardScopeUser, nil, nil, &ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pcscgo.SCardReleaseContext(ctx)

	// 2. List available readers (query size, then fill buffer)
	n, err := pcscgo.SCardListReaders(ctx, "", nil)
	if errors.Is(err, pcscgo.SCardErrorNoReadersAvailable) {
		log.Println("No readers found")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, n)
	if n, err = pcscgo.SCardListReaders(ctx, "", buf); err != nil {
		log.Fatal(err)
	}
	readers := strings.Split(strings.TrimRight(string(buf[:n]), "\x00"), "\x00")

	// 3. Connect to first reader
	var card pcscgo.SCardHandle
	var proto pcscgo.DWord
	err = pcscgo.SCardConnect(ctx, readers[0], pcscgo.SCardShareShared, pcscgo.SCardProtocolAny, &card, &proto)
	if err != nil {
		log.Fatal(err)
	}
	defer pcscgo.SCardDisconnect(card, pcscgo.SCardLeaveCard)

	// 4. Send APDU command (example: SELECT "1PAY.SYS.DDF01")
	selectPSE := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E,
		'1', 'P', 'A', 'Y', '.', 'S', 'Y', 'S', '.', 'D', 'D', 'F', '0', '1', 0x00}
	pci := pcscgo.SCardPCIT1()
	if proto == pcscgo.SCardProtocolT0 {
		pci = pcscgo.SCardPCIT0()
	}
	recv := make([]byte, 256)
	n, err = pcscgo.SCardTransmit(card, pci, selectPSE, nil, recv)
	if err != nil {
		log.Fatal(err)
	}

	// Check response (last 2 bytes = SW1 SW2)
	response := recv[:n]
	if len(response) >= 2 {
		sw1, sw2 := response[len(response)-2], response[len(response)-1]
		log.Printf("SW1SW2: %02X%02X", sw1, sw2)
	}
}
```

## Build Modes

Select **exactly one** backend at compile time via build tags:

| Mode | Build Command | Description |
|------|---------------|-------------|
| **purego** | `go build -tags=pcsc_purego` | Runtime loading via `dlopen`/`dlsym` (no CGO). Cross-compiles easily. |
| **dynamic** | `CGO_ENABLED=1 go build -tags=pcsc_dynamic` | Standard CGO with dynamic linking to `libpcsclite.so`. |
| **static** | `CGO_ENABLED=1 go build -tags=pcsc_static` | CGO with static linking (`libpcsclite.a` + deps). No runtime deps. |

**Only one backend is compiled** — enforced by `mode_guard.go`.

### Purego Mode (Runtime Loading)

In purego mode, load the library at runtime:

```go
import "github.com/ik5/pcscgo"

func main() {
	// Optional: custom path (or set PCSCLITE_LIB_PATH env var)
	if err := pcscgo.Load("/usr/lib/x86_64-linux-gnu/libpcsclite.so.1"); err != nil {
		log.Fatal(err)
	}
	// ... use library normally ...
}
```

Default library paths by OS:
- **Linux**: `libpcsclite.so.1`
- **FreeBSD/OpenBSD/NetBSD**: `libpcsclite.so`

Override via `PCSCLITE_LIB_PATH` environment variable.

## Error Handling

All public functions return `error` instead of raw codes:

```go
var ctx pcscgo.SCardContext
err := pcscgo.SCardEstablishContext(pcscgo.SCardScopeUser, nil, nil, &ctx)
if err != nil {
    var pcscErr *pcscgo.PCSCError
    if errors.As(err, &pcscErr) {
        // pcscErr.Code    - raw PC/SC error code (e.g., 0x8010001D)
        // pcscErr.Message - human-readable description
        fmt.Printf("PC/SC Error: [%#x] %s\n", pcscErr.Code, pcscErr.Message)
    }
    log.Fatal(err)
}
```

Use `errors.Is/As` for specific errors:

```go
if errors.Is(err, pcscgo.SCardErrorNoService) { ... }      // pcscd not running
if errors.Is(err, pcscgo.SCardErrorUnknownReader) { ... }  // reader not found
if errors.Is(err, pcscgo.ErrNotLoaded) { ... }             // purego lib not loaded
```

### Error Types

| Type | Description |
|------|-------------|
| `PCSCError` | Wraps PC/SC error code with message. Implements `error`, `Unwrap() error`, `IsSuccess() bool` |
| `LoadError` | Purego library loading failures (`dlopen`, `dlsym`). Implements `error`, `Unwrap() error` |
| `ErrorCode` | Raw PC/SC code (uint32). Implements `error` interface. |

## Signal Handling

The library supports graceful shutdown via `SCardCancel`:

```go
var ctx pcscgo.SCardContext
// ... establish context ...

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigCh
    pcscgo.SCardCancel(ctx) // Interrupts blocking SCardGetStatusChange
}()

states := []pcscgo.SCardReaderState{...}
err := pcscgo.SCardGetStatusChange(ctx, pcscgo.Infinite, states)
if errors.Is(err, pcscgo.SCardErrorCancelled) {
    // Graceful shutdown
}
```

## Platform Support

| OS | Architectures | Backends |
|----|---------------|----------|
| Linux | amd64, 386, arm64, arm, ... | purego, dynamic, static | ✅ Tested |
| FreeBSD | amd64, 386 | purego, dynamic, static | ✅ Tested |
| OpenBSD | amd64 | purego, dynamic, static | ✅ Tested |
| NetBSD | amd64 | purego, dynamic, static | ✅ Tested |

**macOS and Windows are not supported** — the build fails on any OS outside the table (`libpcscgo_others.go`).

## Memory Layout Warning

Structs `PCSCTLVStructure` through `PINPropertiesStructure` are packed in C (`#pragma pack(1)`). Go adds padding, so **in-memory layout differs from C wire format**.

```go
// WRONG - layout mismatch
var s PCSCTLVStructure
binary.Read(buf, binary.BigEndian, &s) // Use this instead

// NEVER do this:
ptr := unsafe.Pointer(&s) // Layout mismatch!
```

**Rules:**
- Never pass Go struct pointers to C
- Use `encoding/binary` (Read/Write) for wire encoding
- `PCSCTLVStructure.Value` is **always big-endian** (PC/SC Part 10)
- PIN structures use host byte order

## Examples

See [examples/scan](examples/scan) for a complete smart card monitor (like `pcsc_scan`):

```bash
cd examples/scan
make all        # Builds scan, dyn_scan, st_scan
./scan -v       # Run purego version
./dyn_scan -v   # Run dynamic CGO version
```

## Requirements

- Go 1.21+
- `pcscd` running (`systemctl start pcscd`)
- libpcsclite development headers (for CGO builds):
  - Debian/Ubuntu: `libpcsclite-dev`
  - Arch: `pcsc-lite`
  - Fedora: `pcsc-lite-devel`
  - FreeBSD: `devel/pcsc-lite`

## License

BSD-3-Clause — see [LICENSE](LICENSE) for details.

Matches libpcsclite's license for seamless integration.

## References

- [PC/SC Workgroup Specifications](https://pcscworkgroup.com/)
- [pcsc-lite Documentation](https://pcsclite.apdu.fr/)
- [pcsc_scan Source](https://github.com/LudovicRousseau/pcsc-lite/blob/master/src/tools/pcsc_scan.c)
- [purego Library](https://github.com/ebitengine/purego)