// SPDX-License-Identifier: BSD-3-Clause
// Package pcscgo provides a pure Go binding for PC/SC (PC/SC-Lite) smart card middleware.
//
// pcscgo enables Go applications to communicate with smart card readers via the
// PC/SC standard. It supports three linking modes:
//
//   - purego:   Runtime loading via dlopen/dlsym (no CGO required)
//   - dynamic:  CGO with dynamic linking to libpcsclite.so
//   - static:   CGO with static linking (libpcsclite.a + dependencies)
//
// Only one backend is compiled at a time, selected via build tags.
//
// QUICK START
//
//   import "github.com/ik5/pcscgo"
//
//   func main() {
//       // Establish context
//       var ctx pcscgo.SCardContext
//       if err := pcscgo.SCardEstablishContext(pcscgo.SCardScopeUser, nil, nil, &ctx); err != nil {
//           log.Fatal(err)
//       }
//       defer pcscgo.SCardReleaseContext(ctx)
//
//       // List readers
//       readers, _ := pcscgo.SCardListReaders(ctx, "", nil)
//       // ... use readers ...
//   }
//
// BUILD TAGS
//
// Select exactly one backend at compile time:
//
//   // purego (no CGO, runtime loading)
//   go build -tags=pcsc_purego
//
//   // dynamic CGO (shared library)
//   CGO_ENABLED=1 go build -tags=pcsc_dynamic
//
//   // static CGO (static linking)
//   CGO_ENABLED=1 go build -tags=pcsc_static
//
// ARCHITECTURE
//
// The library separates the public API (winscard.go) from backend implementations:
//   - Backend interface (backend.go) defines all PC/SC operations
//   - backend_cgo.go: CGO implementation using #cgo pkg-config
//   - backend_purego.go: purego implementation using github.com/ebitengine/purego
//   - mode_guard.go: Enforces exactly one backend at compile time
//
// ERROR HANDLING
//
// All public functions return error instead of raw error codes:
//
//   var ctx pcscgo.SCardContext
//   err := pcscgo.SCardEstablishContext(pcscgo.SCardScopeUser, nil, nil, &ctx)
//   if err != nil {
//       var pcscErr *pcscgo.PCSCError
//       if errors.As(err, &pcscErr) {
//           // pcscErr.Code   - the raw PC/SC error code (e.g., 0x8010001D)
//           // pcscErr.Message - human-readable message
//       }
//   }
//
// Error types:
//   - PCSCError: Wraps a PC/SC error code with message
//   - LoadError: Purego library loading failures
//   - ErrorCode: Implements error interface for raw codes
//
// Use errors.Is/As for error checking:
//
//   if errors.Is(err, pcscgo.SCardErrorNoService) { ... }
//   if errors.Is(err, pcscgo.ErrNotLoaded) { ... }
//
// PUREGO MODE
//
// In purego mode, the library must be loaded at runtime:
//
//   // Optional: custom library path (or set PCSCLITE_LIB_PATH env var)
//   if err := pcscgo.Load("/usr/lib/x86_64-linux-gnu/libpcsclite.so.1"); err != nil {
//       log.Fatal(err)
//   }
//
// Default library paths by OS:
//   - Linux:        libpcsclite.so.1
//   - FreeBSD/OpenBSD/NetBSD: libpcsclite.so
//   - macOS:        PCSC.framework/PCSC
//
// MEMORY LAYOUT WARNING
//
// The structs from PCSCTLVStructure through PINPropertiesStructure are declared
// in reader.h between "#pragma pack(push, 1)" and "#pragma pack(pop)", so the C
// compiler lays them out with no padding between fields.
//
// Go has no packing attribute. It aligns uint16 to 2 bytes and uint32 to 4, so
// the in-memory layout of these Go structs differs from the C wire layout.
// For example, PCSCTLVStructure is 6 bytes in C (Value at offset 2) but 8 bytes
// in Go (Value at offset 4).
//
// Because of this:
//   - Never cast a byte buffer to these types with unsafe, and never pass a
//     pointer to them to C. The layout does not match.
//   - Do not use unsafe.Sizeof or unsafe.Offsetof on them. They report the
//     padded Go layout. Use binary.Size for the C sizeof.
//   - Convert with encoding/binary (binary.Read / binary.Write), which encodes
//     fixed-size fields back to back with no padding.
//
// Byte order: PCSCTLVStructure.Value is always big-endian (PC/SC part 10).
// The PIN structures use host byte order.
//
// THREAD SAFETY
//
// The library is safe for concurrent use. The purego backend uses mutexes
// to protect library loading and symbol resolution. The CGO backend relies
// on the underlying C library's thread safety.
//
// PLATFORM SUPPORT
//
//   OS           | Architectures                    | Backends           | Status
//   -------------|----------------------------------|--------------------|--------
//   Linux        | amd64, 386, arm64, arm, ...      | purego, dyn, static| ✅ Tested
//   FreeBSD      | amd64, 386                       | purego, dyn, static| ✅ Tested
//   OpenBSD      | amd64                            | purego, dyn, static| ✅ Tested
//   NetBSD       | amd64                            | purego, dyn, static| ✅ Tested
//   macOS        | amd64, arm64                     | purego (framework) | ❓ Untested
//
// Windows is not supported (different PC/SC API).
//
// SEE ALSO
//
//   - examples/scan: Complete smart card monitor (like pcsc_scan)
//   - PC/SC Workgroup: https://pcscworkgroup.com/
//   - pcsc-lite docs: https://pcsclite.apdu.fr/
package pcscgo