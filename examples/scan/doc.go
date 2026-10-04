// Package main demonstrates a complete PC/SC smart card scanner using the pcscgo library.
//
// This example implements a smart card reader monitor similar to the standard
// pcsc_scan utility that comes with pcsc-lite. It demonstrates proper usage of
// the pcscgo library including:
//
//   - Establishing a context with the PC/SC resource manager (pcscd)
//   - Enumerating smart card readers
//   - Monitoring readers for state changes (card insert/remove, reader connect/disconnect)
//   - Proper signal handling using SCardCancel to interrupt blocking calls
//   - Handling PnP notifications for reader hot-plug detection
//   - Memory pinning for CGO backends
//
// BUILD MODES
//
// The scanner can be built in three different linking modes:
//
//   1. purego (runtime loading):     go build -tags=pcsc_purego -o scan
//   2. dynamic CGO (shared library): CGO_ENABLED=1 go build -tags=pcsc_dynamic -o dyn_scan
//   3. static CGO (static linking):  CGO_ENABLED=1 go build -tags=pcsc_static -o st_scan
//
// The Makefile provides convenient targets: make scan, make dyn_scan, make st_scan,
// make all, make clean, make test.
//
// RUNNING
//
//   ./scan              # purego with default library path
//   ./scan -lib /path   # purego with custom library path
//   ./dyn_scan          # dynamic CGO
//   ./st_scan           # static CGO
//
//   Flags:
//     -lib string       # Path to libpcsclite.so (purego only)
//     -timeout duration # -1=infinite (default), 0=1s poll, >0=custom ms
//     -v                # Verbose output
//
// ARCHITECTURE
//
// The scanner uses the standard PC/SC event-driven model:
//
//  1. SCardEstablishContext - connect to pcscd daemon
//  2. SCardListReaders    - enumerate available readers
//  3. SCardGetStatusChange - block until reader state changes
//  4. Process events, update state, repeat
//
// KEY DESIGN POINTS
//
// Signal Handling:
//
// The blocking SCardGetStatusChange call is interrupted using SCardCancel
// from a signal handler (SIGINT, SIGTERM). This is the same technique
// used by pcsc_scan and is the only reliable way to interrupt the
// blocking C call. The signal handler calls SCardCancel(ctxHandle),
// which causes SCardGetStatusChange to return SCardErrorCancelled,
// which we handle as a graceful shutdown.
//
// Memory Safety (CGO):
//
// For CGO backends, Go memory containing pointers passed to C must be
// pinned to prevent the garbage collector from moving it during the C call.
// We use runtime.Pinner to pin the states slice and its pointer fields
// (Reader, UserData) during the SCardGetStatusChange call.
//
// State Management:
//
// The scanner tracks per-reader state to detect transitions:
//   - Reader connection/disconnection (SCardStateUnavailable)
//   - Card insertion/removal (SCardStatePresent/Empty)
//   - PnP notifications for hot-plug (\\?PnP?\Notification)
//
// The states are initialized with SCardStateUnaware, which tells
// SCardGetStatusChange to return the current state immediately on
// the first call. Subsequent calls use the previous EventState as
// CurrentState to detect only changes.
//
// PnP Hot-Plug:
//
// The special reader name "\\?PnP?\Notification" receives notifications
// when readers are added or removed. On receiving such a notification,
// the scanner re-enumerates the reader list and rebuilds its internal
// state array.
//
// BUILD TAGS
//
// The three backends are selected via build tags:
//   - pcsc_purego:   Uses github.com/ebitengine/purego for runtime dlopen/dlsym
//   - pcsc_dynamic:  CGO with #cgo pkg-config: libpcsclite
//   - pcsc_static:   CGO with #cgo pkg-config: --static libpcsclite
//
// Only one backend is compiled at a time (enforced by mode_guard.go in pcscgo).
//
// ENVIRONMENT
//
// The purego backend can be configured via environment variable:
//   PCSCLITE_LIB_PATH=/path/to/libpcsclite.so
//
// If not set, defaults are used:
//   Linux:        libpcsclite.so.1
//   FreeBSD/OpenBSD/NetBSD: libpcsclite.so
//   macOS:        PCSC.framework/PCSC
//
// EXAMPLE OUTPUT
//
//   === PC/SC Smart Card Scanner ===
//   Mode: purego (runtime library loading)
//   Library: libpcsclite.so.1 (default)
//
//   Found 1 reader(s):
//     0: ACS ACR39U ICC Reader 01 00
//
//   Monitoring for card/reader events... (Press Ctrl+C to stop)
//
//   [ACS ACR39U ICC Reader 01 00] CARD INSERTED
//     ATR: 3B 8F 80 01 80 4F 0C A0 00 00 03 06 03 00 01 00 00 00 00 6A
//   [ACS ACR39U ICC Reader 01 00] CARD REMOVED
//   [ACS ACR39U ICC Reader 01 00] READER DISCONNECTED (unavailable)
//   READER REMOVED: (reader disconnected via PnP)
//   ^C
//   Shutting down...
//   Released context
//
// See the README.md for more details on building and running.
package main