package pcscgo

import "unsafe"

// Backend defines the PC/SC operations implemented by each binding mode.
// This interface allows tests to inject mock implementations.
type Backend interface {
	EstablishContext(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) error
	ReleaseContext(ctx SCardContext) error
	IsValidContext(ctx SCardContext) error
	Connect(ctx SCardContext, reader LPCStr, shareMode, preferredProtocols DWord, card *SCardHandle, activeProtocol LPDWord) error
	Reconnect(card SCardHandle, shareMode, preferredProtocols, initialization DWord, activeProtocol LPDWord) error
	Disconnect(card SCardHandle, disposition DWord) error
	BeginTransaction(card SCardHandle) error
	EndTransaction(card SCardHandle, disposition DWord) error
	Status(card SCardHandle, readerName LPStr, readerLen, state, protocol LPDWord, atr LPByte, atrLen LPDWord) error
	GetStatusChange(ctx SCardContext, timeout DWord, states *SCardReaderState, n DWord) error
	Control(card SCardHandle, controlCode DWord, send LPCByte, sendLen DWord, recv LPByte, recvLen DWord, returned LPDWord) error
	Transmit(card SCardHandle, sendPci LPCSCardIORequest, send LPCByte, sendLen DWord, recvPci LPSCardIORequest, recv LPByte, recvLen LPDWord) error
	ListReaderGroups(ctx SCardContext, groups LPStr, groupsLen LPDWord) error
	ListReaders(ctx SCardContext, groups LPCStr, readers LPStr, readersLen LPDWord) error
	FreeMemory(ctx SCardContext, mem unsafe.Pointer) error
	Cancel(ctx SCardContext) error
	GetAttrib(card SCardHandle, attrID DWord, attr LPByte, attrLen LPDWord) error
	SetAttrib(card SCardHandle, attrID DWord, attr LPCByte, attrLen DWord) error
	PCIT0() *SCardIORequest
	PCIT1() *SCardIORequest
	PCIRaw() *SCardIORequest
	StringifyError(rc Long) string
	// LibraryPath returns the path to the PC/SC library being used.
	// For CGO backends this returns an empty string.
	LibraryPath() string
}

// backend is the active backend implementation.
// It is set by the init() function in each backend_*.go file.
var backend Backend

// SetBackend replaces the active backend (for testing).
// This should only be called from tests.
func SetBackend(b Backend) {
	backend = b
}

// getBackend returns the active backend, initializing it if needed.
func getBackend() Backend {
	if backend == nil {
		panic("pcscgo: no backend initialized - import a backend package (pcsc_static, pcsc_dynamic, or pcsc_purego)")
	}
	return backend
}
