// SPDX-License-Identifier: BSD-3-Clause

package pcscgo

import (
	"runtime"
	"syscall"
	"unsafe"
)

// SCardEstablishContext establishes a context with the PC/SC resource manager.
//
// The scope parameter defines the scope of the context:
//   - SCardScopeUser:   user scope
//   - SCardScopeTerminal: terminal scope
//   - SCardScopeSystem: system scope
//   - SCardScopeGlobal: global scope
//
// The r1 and r2 parameters are reserved for future use and should be nil.
// On success, ctx is set to the established context handle.
//
// This function corresponds to the PC/SC SCardEstablishContext function.
func SCardEstablishContext(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) error {
	return getBackend().EstablishContext(scope, r1, r2, ctx)
}

// SCardReleaseContext releases a context handle obtained from SCardEstablishContext.
//
// After this call, the context handle is no longer valid.
// This function corresponds to the PC/SC SCardReleaseContext function.
func SCardReleaseContext(ctx SCardContext) error {
	return getBackend().ReleaseContext(ctx)
}

// SCardIsValidContext checks whether a context handle is valid.
//
// Returns nil if the context is valid, otherwise returns an error.
// This function corresponds to the PC/SC SCardIsValidContext function.
func SCardIsValidContext(ctx SCardContext) error {
	return getBackend().IsValidContext(ctx)
}

// SCardConnect establishes a connection to a smart card reader.
//
// Parameters:
//   - ctx:       context from SCardEstablishContext
//   - reader:    name of the reader (as returned by SCardListReaders)
//   - shareMode: sharing mode (SCardShareExclusive, SCardShareShared, SCardShareDirect)
//   - preferredProtocols: preferred protocols (SCardProtocolT0, SCardProtocolT1, SCardProtocolRaw, or SCardProtocolAny)
//   - card:      on success, set to the card handle
//   - activeProtocol: on success, set to the negotiated protocol
//
// Returns an error if the connection cannot be established.
// This function corresponds to the PC/SC SCardConnect function.
func SCardConnect(
	ctx SCardContext, reader string, shareMode DWord, preferredProtocols DWord, card *SCardHandle,
	activeProtocol LPDWord,
) error {
	r, err := syscall.BytePtrFromString(reader)
	if err != nil { // embedded NUL
		return SCardErrorInvalidParameter
	}

	err = getBackend().Connect(ctx, r, shareMode, preferredProtocols, card, activeProtocol)
	runtime.KeepAlive(r)

	return err
}

// SCardReconnect re-establishes a connection to a card after the connection has been lost.
//
// Parameters:
//   - card:             card handle from SCardConnect
//   - shareMode:        new sharing mode
//   - preferredProtocols: new preferred protocols
//   - initialization:   card initialization type (SCardLeaveCard, SCardResetCard, SCardUnpowerCard, SCardEjectCard)
//   - activeProtocol:   on success, set to the negotiated protocol
//
// This function corresponds to the PC/SC SCardReconnect function.
func SCardReconnect(
	card SCardHandle, shareMode DWord, preferredProtocols DWord, initialization DWord, activeProtocol LPDWord,
) error {
	return getBackend().Reconnect(card, shareMode, preferredProtocols, initialization, activeProtocol)
}

// SCardDisconnect terminates a connection to a smart card.
//
// Parameters:
//   - card:       card handle from SCardConnect
//   - disposition: action on disconnect (SCardLeaveCard, SCardResetCard, SCardUnpowerCard, SCardEjectCard)
//
// This function corresponds to the PC/SC SCardDisconnect function.
func SCardDisconnect(card SCardHandle, disposition DWord) error {
	return getBackend().Disconnect(card, disposition)
}

// SCardBeginTransaction starts a transaction on a card connection.
//
// A transaction ensures exclusive access to the card for a sequence of operations.
// Must be paired with SCardEndTransaction.
// This function corresponds to the PC/SC SCardBeginTransaction function.
func SCardBeginTransaction(card SCardHandle) error {
	return getBackend().BeginTransaction(card)
}

// SCardEndTransaction ends a transaction on a card connection.
//
// Parameters:
//   - card:        card handle from SCardConnect
//   - disposition: action on end (SCardLeaveCard, SCardResetCard, SCardUnpowerCard, SCardEjectCard)
//
// This function corresponds to the PC/SC SCardEndTransaction function.
func SCardEndTransaction(card SCardHandle, disposition DWord) error {
	return getBackend().EndTransaction(card, disposition)
}

// bufPtr returns a pointer to the first byte of b, or nil when b is empty.
// An empty buffer reaches C as NULL, which asks the library for the needed length.
func bufPtr(b []byte) *Byte {
	if len(b) == 0 {
		return nil
	}
	return &b[0]
}

// SCardStatus retrieves the current status of a smart card.
//
// Parameters:
//   - card:       card handle from SCardConnect
//   - readerName: buffer to receive the reader name (may be empty to query length)
//   - atr:        buffer to receive the ATR (may be empty to query length)
//
// Returns:
//   - readerLen:  actual/required length of reader name
//   - state:      card state (SCardUnknown, SCardAbsent, SCardPresent, etc.)
//   - protocol:   active protocol (SCardProtocolT0, SCardProtocolT1, etc.)
//   - atrLen:     actual/required length of ATR
//   - error:      nil on success
//
// Pass empty slices for readerName and/or atr to query the required buffer sizes.
// This function corresponds to the PC/SC SCardStatus function.
func SCardStatus(card SCardHandle, readerName, atr []byte) (DWord, DWord, DWord, DWord, error) {
	readerLen, atrLen := DWord(len(readerName)), DWord(len(atr))

	var state, protocol DWord

	err := getBackend().Status(card, bufPtr(readerName), &readerLen, &state, &protocol, bufPtr(atr), &atrLen)
	runtime.KeepAlive(readerName)
	runtime.KeepAlive(atr)

	return readerLen, state, protocol, atrLen, err
}

// SCardGetStatusChange blocks until a reader state changes or timeout expires.
//
// Parameters:
//   - ctx:     context from SCardEstablishContext
//   - timeout: maximum time to wait (in milliseconds); use Infinite for no timeout
//   - states:  slice of reader states to monitor; each entry's CurrentState is the
//     expected state, and on return EventState contains the new state
//
// The function updates the EventState and ATR fields of each entry in place.
// This function corresponds to the PC/SC SCardGetStatusChange function.
func SCardGetStatusChange(ctx SCardContext, timeout DWord, states []SCardReaderState) error {
	var states0 *SCardReaderState
	if len(states) > 0 {
		states0 = &states[0]
	}

	// cgo forbids passing Go memory that holds unpinned Go pointers.
	var pinner runtime.Pinner
	defer pinner.Unpin()
	for i := range states {
		if states[i].Reader != nil {
			pinner.Pin(states[i].Reader)
		}

		if states[i].UserData != nil {
			pinner.Pin(states[i].UserData)
		}
	}

	err := getBackend().GetStatusChange(ctx, timeout, states0, DWord(len(states)))
	runtime.KeepAlive(states)

	return err
}

// SCardControl sends a control command directly to the reader driver (IFD).
//
// Parameters:
//   - card:        card handle from SCardConnect
//   - controlCode: control code (e.g., SCardCtlCode(3400) for FEATURE_VERIFY_PIN_START)
//   - send:        input buffer
//   - recv:        output buffer (may be empty to query length)
//
// Returns:
//   - n:     number of bytes written to recv
//   - error: nil on success
//
// This function corresponds to the PC/SC SCardControl function.
func SCardControl(card SCardHandle, controlCode DWord, send, recv []byte) (DWord, error) {
	var n DWord

	err := getBackend().Control(card, controlCode, bufPtr(send), DWord(len(send)), bufPtr(recv), DWord(len(recv)), &n)

	runtime.KeepAlive(send)
	runtime.KeepAlive(recv)

	return n, err
}

// SCardTransmit sends an APDU to the smart card and receives the response.
//
// Parameters:
//   - card:      card handle from SCardConnect
//   - sendPci:   send protocol control info (SCardPCIT0, SCardPCIT1, or SCardPCIRaw)
//   - send:      command APDU
//   - recvPci:   receive protocol control info (may be nil)
//   - recv:      response buffer (may be empty to query length)
//
// Returns:
//   - n:     number of bytes written to recv
//   - error: nil on success
//
// This function corresponds to the PC/SC SCardTransmit function.
func SCardTransmit(
	card SCardHandle, sendPci LPCSCardIORequest, send []byte, recvPci LPSCardIORequest, recv []byte,
) (DWord, error) {
	n := DWord(len(recv))
	err := getBackend().Transmit(card, sendPci, bufPtr(send), DWord(len(send)), recvPci, bufPtr(recv), &n)

	runtime.KeepAlive(send)
	runtime.KeepAlive(recv)

	return n, err
}

// SCardListReaderGroups enumerates reader groups.
//
// Parameters:
//   - ctx:    context from SCardEstablishContext
//   - groups: buffer to receive multi-string of group names (may be empty to query length)
//
// Returns:
//   - n:     actual/required length of multi-string
//   - error: nil on success
//
// A multi-string is a sequence of NUL-terminated strings, terminated by an additional NUL.
// This function corresponds to the PC/SC SCardListReaderGroups function.
func SCardListReaderGroups(ctx SCardContext, groups []byte) (DWord, error) {
	n := DWord(len(groups))
	err := getBackend().ListReaderGroups(ctx, bufPtr(groups), &n)

	runtime.KeepAlive(groups)

	return n, err
}

// SCardListReaders enumerates smart card readers.
//
// Parameters:
//   - ctx:      context from SCardEstablishContext
//   - groups:   reader group filter (multi-string, e.g., "Group1\x00Group2\x00\x00"); "" means all readers
//   - readers:  buffer to receive multi-string of reader names (may be empty to query length)
//
// Returns:
//   - n:     actual/required length of multi-string
//   - error: nil on success
//
// This function corresponds to the PC/SC SCardListReaders function.
func SCardListReaders(ctx SCardContext, groups string, readers []byte) (DWord, error) {
	var g []byte
	if groups != "" {
		g = []byte(groups + "\x00\x00")
	}

	n := DWord(len(readers))
	err := getBackend().ListReaders(ctx, bufPtr(g), bufPtr(readers), &n)

	runtime.KeepAlive(g)
	runtime.KeepAlive(readers)

	return n, err
}

// SCardFreeMemory frees memory allocated by the PC/SC resource manager.
//
// Parameters:
//   - ctx: context from SCardEstablishContext
//   - mem: pointer returned by a previous call (e.g., SCardListReaders with auto-allocate)
//
// This function corresponds to the PC/SC SCardFreeMemory function.
func SCardFreeMemory(ctx SCardContext, mem unsafe.Pointer) error {
	return getBackend().FreeMemory(ctx, mem)
}

// SCardCancel cancels all pending operations on a context.
//
// This function corresponds to the PC/SC SCardCancel function.
func SCardCancel(ctx SCardContext) error {
	return getBackend().Cancel(ctx)
}

// SCardGetAttrib retrieves a reader or card attribute.
//
// Parameters:
//   - card:    card handle from SCardConnect
//   - attrID:  attribute ID (e.g., SCardAttrVendorName, SCardAttrATRString, SCardAttrDeviceFriendlyName)
//   - attr:    buffer to receive attribute value (may be empty to query length)
//
// Returns:
//   - n:     actual/required length of attribute value
//   - error: nil on success
//
// This function corresponds to the PC/SC SCardGetAttrib function.
func SCardGetAttrib(card SCardHandle, attrID DWord, attr []byte) (DWord, error) {
	n := DWord(len(attr))
	err := getBackend().GetAttrib(card, attrID, bufPtr(attr), &n)

	runtime.KeepAlive(attr)

	return n, err
}

// SCardSetAttrib sets a reader or card attribute.
//
// Parameters:
//   - card:    card handle from SCardConnect
//   - attrID:  attribute ID
//   - attr:    attribute value to set
//
// This function corresponds to the PC/SC SCardSetAttrib function.
func SCardSetAttrib(card SCardHandle, attrID DWord, attr []byte) error {
	err := getBackend().SetAttrib(card, attrID, bufPtr(attr), DWord(len(attr)))

	runtime.KeepAlive(attr)

	return err
}

// SCardPCIT0 returns the T=0 protocol control information (PCI).
// The returned pointer refers to library memory and must be treated as read-only.
// This function corresponds to the PC/SC g_rgSCardT0Pci symbol.
func SCardPCIT0() LPCSCardIORequest {
	return getBackend().PCIT0()
}

// SCardPCIT1 returns the T=1 protocol control information (PCI).
// The returned pointer refers to library memory and must be treated as read-only.
// This function corresponds to the PC/SC g_rgSCardT1Pci symbol.
func SCardPCIT1() LPCSCardIORequest {
	return getBackend().PCIT1()
}

// SCardPCIRaw returns the RAW protocol control information (PCI).
// The returned pointer refers to library memory and must be treated as read-only.
// This function corresponds to the PC/SC g_rgSCardRawPci symbol.
func SCardPCIRaw() LPCSCardIORequest {
	return getBackend().PCIRaw()
}

// PCSCStringifyError gets an error code and translates it into a string.
// This function corresponds to the PC/SC pcsc_stringify_error function.
func PCSCStringifyError(rc ErrorCode) string {
	return getBackend().StringifyError(Long(rc))
}

// PCSCLibraryPath returns the library path that will be used by the backend.
// For CGO backends, this returns an empty string.
// The path can be overridden by setting the PCSCLITE_LIB_PATH environment variable (purego only).
func PCSCLibraryPath() string {
	return getBackend().LibraryPath()
}
