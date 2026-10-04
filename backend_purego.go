//go:build pcsc_purego

package pcscgo

import (
	"os"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	mu     sync.RWMutex
	handle uintptr // 0 means not loaded
)

var (
	fnSCardEstablishContext func(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) Long
	fnPCSCStringifyError    func(rc Long) uintptr
	fnSCardReleaseContext   func(ctx SCardContext) Long
	fnSCardIsValidContext   func(ctx SCardContext) Long
	fnSCardConnect          func(ctx SCardContext, reader LPCStr, shareMode, preferredProtocols DWord,
		card *SCardHandle, activeProtocol *DWord,
	) Long
	fnSCardReconnect func(
		card SCardHandle, shareMode DWord, preferredProtocols DWord, initialization DWord, activeProtocol LPDWord,
	) Long
	fnSCardDisconnect       func(card SCardHandle, disposition DWord) Long
	fnSCardBeginTransaction func(card SCardHandle) Long
	fnSCardEndTransaction   func(card SCardHandle, disposition DWord) Long
	fnSCardStatus           func(
		card SCardHandle, readerName LPStr, readerLen, state, protocol LPDWord, atr LPByte, atrLen LPDWord,
	) Long
	fnSCardGetStatusChange func(ctx SCardContext, timeout DWord, states *SCardReaderState, n DWord) Long
	fnSCardControl         func(
		card SCardHandle, controlCode DWord, send LPCByte, sendLen DWord, recv LPByte, recvLen DWord, returned LPDWord,
	) Long
	fnSCardTransmit func(
		card SCardHandle, sendPci LPCSCardIORequest, send LPCByte, sendLen DWord,
		recvPci LPSCardIORequest, recv LPByte, recvLen LPDWord,
	) Long
	fnSCardListReaderGroups func(ctx SCardContext, groups LPStr, groupsLen LPDWord) Long
	fnSCardListReaders      func(ctx SCardContext, groups LPCStr, readers LPStr, readersLen LPDWord) Long
	fnSCardFreeMemory       func(ctx SCardContext, mem unsafe.Pointer) Long
	fnSCardCancel           func(ctx SCardContext) Long
	fnSCardGetAttrib        func(card SCardHandle, attrID DWord, attr LPByte, attrLen LPDWord) Long
	fnSCardSetAttrib        func(card SCardHandle, attrID DWord, attr LPCByte, attrLen DWord) Long
)

var pciT0, pciT1, pciRaw uintptr

var syms = []struct {
	name string
	dst  *uintptr
	fn   any
}{
	{dst: &pciT0, name: "g_rgSCardT0Pci"},
	{dst: &pciT1, name: "g_rgSCardT1Pci"},
	{dst: &pciRaw, name: "g_rgSCardRawPci"},
	{fn: &fnSCardEstablishContext, name: "SCardEstablishContext"},
	{fn: &fnPCSCStringifyError, name: "pcsc_stringify_error"},
	{fn: &fnSCardReleaseContext, name: "SCardReleaseContext"},
	{fn: &fnSCardIsValidContext, name: "SCardIsValidContext"},
	{fn: &fnSCardConnect, name: "SCardConnect"},
	{fn: &fnSCardReconnect, name: "SCardReconnect"},
	{fn: &fnSCardDisconnect, name: "SCardDisconnect"},
	{fn: &fnSCardBeginTransaction, name: "SCardBeginTransaction"},
	{fn: &fnSCardEndTransaction, name: "SCardEndTransaction"},
	{fn: &fnSCardStatus, name: "SCardStatus"},
	{fn: &fnSCardGetStatusChange, name: "SCardGetStatusChange"},
	{fn: &fnSCardControl, name: "SCardControl"},
	{fn: &fnSCardTransmit, name: "SCardTransmit"},
	{fn: &fnSCardListReaderGroups, name: "SCardListReaderGroups"},
	{fn: &fnSCardListReaders, name: "SCardListReaders"},
	{fn: &fnSCardFreeMemory, name: "SCardFreeMemory"},
	{fn: &fnSCardCancel, name: "SCardCancel"},
	{fn: &fnSCardGetAttrib, name: "SCardGetAttrib"},
	{fn: &fnSCardSetAttrib, name: "SCardSetAttrib"},
}

type puregoBackend struct {
	loadOnce   sync.Once
	loadErr    error
	loadedPath string
}

func (b *puregoBackend) load() error {
	b.loadOnce.Do(func() {
		path := defaultLibraryPath()
		b.loadedPath = path
		b.loadErr = Load(path)
	})
	return b.loadErr
}

func (b *puregoBackend) withLoaded(fn func() error) error {
	if err := b.load(); err != nil {
		return err
	}
	return fn()
}

func (b *puregoBackend) EstablishContext(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardEstablishContext)
		if !ok {
			return &LoadError{Op: "SCardEstablishContext", Err: ErrNotLoaded}
		}
		return toError(fn(scope, r1, r2, ctx))
	})
}

func (b *puregoBackend) ReleaseContext(ctx SCardContext) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardReleaseContext)
		if !ok {
			return &LoadError{Op: "SCardReleaseContext", Err: ErrNotLoaded}
		}
		return toError(fn(ctx))
	})
}

func (b *puregoBackend) IsValidContext(ctx SCardContext) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardIsValidContext)
		if !ok {
			return &LoadError{Op: "SCardIsValidContext", Err: ErrNotLoaded}
		}
		return toError(fn(ctx))
	})
}

func (b *puregoBackend) Connect(ctx SCardContext, reader LPCStr, shareMode, preferredProtocols DWord, card *SCardHandle, activeProtocol *DWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardConnect)
		if !ok {
			return &LoadError{Op: "SCardConnect", Err: ErrNotLoaded}
		}
		return toError(fn(ctx, reader, shareMode, preferredProtocols, card, activeProtocol))
	})
}

func (b *puregoBackend) Reconnect(card SCardHandle, shareMode, preferredProtocols, initialization DWord, activeProtocol LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardReconnect)
		if !ok {
			return &LoadError{Op: "SCardReconnect", Err: ErrNotLoaded}
		}
		return toError(fn(card, shareMode, preferredProtocols, initialization, activeProtocol))
	})
}

func (b *puregoBackend) Disconnect(card SCardHandle, disposition DWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardDisconnect)
		if !ok {
			return &LoadError{Op: "SCardDisconnect", Err: ErrNotLoaded}
		}
		return toError(fn(card, disposition))
	})
}

func (b *puregoBackend) BeginTransaction(card SCardHandle) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardBeginTransaction)
		if !ok {
			return &LoadError{Op: "SCardBeginTransaction", Err: ErrNotLoaded}
		}
		return toError(fn(card))
	})
}

func (b *puregoBackend) EndTransaction(card SCardHandle, disposition DWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardEndTransaction)
		if !ok {
			return &LoadError{Op: "SCardEndTransaction", Err: ErrNotLoaded}
		}
		return toError(fn(card, disposition))
	})
}

func (b *puregoBackend) Status(card SCardHandle, readerName LPStr, readerLen, state, protocol LPDWord, atr LPByte, atrLen LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardStatus)
		if !ok {
			return &LoadError{Op: "SCardStatus", Err: ErrNotLoaded}
		}
		return toError(fn(card, readerName, readerLen, state, protocol, atr, atrLen))
	})
}

func (b *puregoBackend) GetStatusChange(ctx SCardContext, timeout DWord, states *SCardReaderState, n DWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardGetStatusChange)
		if !ok {
			return &LoadError{Op: "SCardGetStatusChange", Err: ErrNotLoaded}
		}
		return toError(fn(ctx, timeout, states, n))
	})
}

func (b *puregoBackend) Control(card SCardHandle, controlCode DWord, send LPCByte, sendLen DWord, recv LPByte, recvLen DWord, returned LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardControl)
		if !ok {
			return &LoadError{Op: "SCardControl", Err: ErrNotLoaded}
		}
		return toError(fn(card, controlCode, send, sendLen, recv, recvLen, returned))
	})
}

func (b *puregoBackend) Transmit(card SCardHandle, sendPci LPCSCardIORequest, send LPCByte, sendLen DWord, recvPci LPSCardIORequest, recv LPByte, recvLen LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardTransmit)
		if !ok {
			return &LoadError{Op: "SCardTransmit", Err: ErrNotLoaded}
		}
		return toError(fn(card, sendPci, send, sendLen, recvPci, recv, recvLen))
	})
}

func (b *puregoBackend) ListReaderGroups(ctx SCardContext, groups LPStr, groupsLen LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardListReaderGroups)
		if !ok {
			return &LoadError{Op: "SCardListReaderGroups", Err: ErrNotLoaded}
		}
		return toError(fn(ctx, groups, groupsLen))
	})
}

func (b *puregoBackend) ListReaders(ctx SCardContext, groups LPCStr, readers LPStr, readersLen LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardListReaders)
		if !ok {
			return &LoadError{Op: "SCardListReaders", Err: ErrNotLoaded}
		}
		return toError(fn(ctx, groups, readers, readersLen))
	})
}

func (b *puregoBackend) FreeMemory(ctx SCardContext, mem unsafe.Pointer) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardFreeMemory)
		if !ok {
			return &LoadError{Op: "SCardFreeMemory", Err: ErrNotLoaded}
		}
		return toError(fn(ctx, mem))
	})
}

func (b *puregoBackend) Cancel(ctx SCardContext) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardCancel)
		if !ok {
			return &LoadError{Op: "SCardCancel", Err: ErrNotLoaded}
		}
		return toError(fn(ctx))
	})
}

func (b *puregoBackend) GetAttrib(card SCardHandle, attrID DWord, attr LPByte, attrLen LPDWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardGetAttrib)
		if !ok {
			return &LoadError{Op: "SCardGetAttrib", Err: ErrNotLoaded}
		}
		return toError(fn(card, attrID, attr, attrLen))
	})
}

func (b *puregoBackend) SetAttrib(card SCardHandle, attrID DWord, attr LPCByte, attrLen DWord) error {
	return b.withLoaded(func() error {
		fn, ok := fnOf(&fnSCardSetAttrib)
		if !ok {
			return &LoadError{Op: "SCardSetAttrib", Err: ErrNotLoaded}
		}
		return toError(fn(card, attrID, attr, attrLen))
	})
}

func (b *puregoBackend) PCIT0() *SCardIORequest {
	if b.load() != nil {
		return nil
	}
	return symPtr[SCardIORequest](&pciT0)
}

func (b *puregoBackend) PCIT1() *SCardIORequest {
	if b.load() != nil {
		return nil
	}
	return symPtr[SCardIORequest](&pciT1)
}

func (b *puregoBackend) PCIRaw() *SCardIORequest {
	if b.load() != nil {
		return nil
	}
	return symPtr[SCardIORequest](&pciRaw)
}

func (b *puregoBackend) StringifyError(rc Long) string {
	if b.load() != nil {
		return ""
	}
	fn, ok := fnOf(&fnPCSCStringifyError)
	if !ok {
		return ""
	}
	return goString(fn(rc))
}

func (b *puregoBackend) LibraryPath() string {
	if b.loadedPath != "" {
		return b.loadedPath
	}
	return defaultLibraryPath()
}

func bind(h uintptr) error {
	var (
		err       error
		addr      uintptr
		addresses = make([]uintptr, len(syms))
	)

	for i, sym := range syms {
		addr, err = purego.Dlsym(h, sym.name)
		if err != nil {
			return &LoadError{Op: "dlsym " + sym.name, Err: err}
		}

		addresses[i] = addr
	}

	for i, sym := range syms {
		switch {
		case sym.fn != nil:
			purego.RegisterFunc(sym.fn, addresses[i])
		case sym.dst != nil:
			*sym.dst = addresses[i]
		}
	}

	return nil
}

// Load opens the PC/SC-Lite shared library at path. path can be an
// absolute file name or a name the dynamic loader searches for.
func Load(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if handle != 0 {
		return &LoadError{Op: "load", Err: ErrAlreadyLoaded}
	}

	h, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return &LoadError{Op: "dlopen", Err: err}
	}

	if err := bind(h); err != nil {
		_ = purego.Dlclose(h)
		return err
	}

	handle = h
	return nil
}

// goString copies the NULL-terminated C string at address p into a Go string.
// p must point to memory outside the Go heap (here: owned by the library).
func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&p)) // keeps go vet quiet, as in symPtr
	n := 0
	for *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n)) // string() copies the bytes
}

func fnOf[F any](fn *F) (F, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if handle == 0 {
		var zero F
		return zero, false
	}
	return *fn, true
}

func symPtr[T any](addr *uintptr) *T {
	mu.RLock()
	defer mu.RUnlock()
	if handle == 0 {
		return nil
	}
	return (*T)(*(*unsafe.Pointer)(unsafe.Pointer(addr)))
}

// defaultLibraryPath returns the default library path for the current platform.
// It can be overridden by setting the PCSCLITE_LIB_PATH environment variable.
func defaultLibraryPath() string {
	if v := os.Getenv("PCSCLITE_LIB_PATH"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "linux":
		return "libpcsclite.so.1"
	case "freebsd", "openbsd", "netbsd":
		return "libpcsclite.so"
	case "darwin":
		return "PCSC.framework/PCSC"
	default:
		return "libpcsclite.so"
	}
}

var puregoBackendInstance = &puregoBackend{}

func init() {
	backend = puregoBackendInstance
}
