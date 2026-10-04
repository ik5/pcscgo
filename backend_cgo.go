// SPDX-License-Identifier: BSD-3-Clause
//go:build (pcsc_static || pcsc_dynamic) && cgo

package pcscgo

/*
#include <pcsclite.h>
#include <winscard.h>
*/
import "C"

import "unsafe"

type cgoBackend struct{}

func (cgoBackend) EstablishContext(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) error {
	return toError(Long(C.SCardEstablishContext(
		C.DWORD(scope),
		C.LPCVOID(r1),
		C.LPCVOID(r2),
		(*C.SCARDCONTEXT)(unsafe.Pointer(ctx)),
	)))
}

func (cgoBackend) ReleaseContext(ctx SCardContext) error {
	return toError(Long(C.SCardReleaseContext(C.SCARDCONTEXT(ctx))))
}

func (cgoBackend) IsValidContext(ctx SCardContext) error {
	return toError(Long(C.SCardIsValidContext(C.SCARDCONTEXT(ctx))))
}

func (cgoBackend) Connect(ctx SCardContext, reader LPCStr, shareMode, preferredProtocols DWord, card *SCardHandle, activeProtocol *DWord) error {
	return toError(Long(C.SCardConnect(
		C.SCARDCONTEXT(ctx),
		C.LPCSTR(unsafe.Pointer(reader)),
		C.DWORD(shareMode),
		C.DWORD(preferredProtocols),
		(*C.SCARDHANDLE)(unsafe.Pointer(card)),
		C.LPDWORD(unsafe.Pointer(activeProtocol)),
	)))
}

func (cgoBackend) Reconnect(card SCardHandle, shareMode, preferredProtocols, initialization DWord, activeProtocol LPDWord) error {
	return toError(Long(C.SCardReconnect(
		C.SCARDHANDLE(card),
		C.DWORD(shareMode),
		C.DWORD(preferredProtocols),
		C.DWORD(initialization),
		C.LPDWORD(unsafe.Pointer(activeProtocol)),
	)))
}

func (cgoBackend) Disconnect(card SCardHandle, disposition DWord) error {
	return toError(Long(C.SCardDisconnect(
		C.SCARDHANDLE(card),
		C.DWORD(disposition),
	)))
}

func (cgoBackend) BeginTransaction(card SCardHandle) error {
	return toError(Long(C.SCardBeginTransaction(C.SCARDHANDLE(card))))
}

func (cgoBackend) EndTransaction(card SCardHandle, disposition DWord) error {
	return toError(Long(C.SCardEndTransaction(
		C.SCARDHANDLE(card),
		C.DWORD(disposition),
	)))
}

func (cgoBackend) Status(card SCardHandle, readerName LPStr, readerLen, state, protocol LPDWord, atr LPByte, atrLen LPDWord) error {
	return toError(Long(C.SCardStatus(
		C.SCARDHANDLE(card),
		C.LPSTR(unsafe.Pointer(readerName)),
		C.LPDWORD(unsafe.Pointer(readerLen)),
		C.LPDWORD(unsafe.Pointer(state)),
		C.LPDWORD(unsafe.Pointer(protocol)),
		C.LPBYTE(unsafe.Pointer(atr)),
		C.LPDWORD(unsafe.Pointer(atrLen)),
	)))
}

func (cgoBackend) GetStatusChange(ctx SCardContext, timeout DWord, states *SCardReaderState, n DWord) error {
	return toError(Long(C.SCardGetStatusChange(
		C.SCARDCONTEXT(ctx),
		C.DWORD(timeout),
		(*C.SCARD_READERSTATE)(unsafe.Pointer(states)),
		C.DWORD(n),
	)))
}

func (cgoBackend) Control(card SCardHandle, controlCode DWord, send LPCByte, sendLen DWord, recv LPByte, recvLen DWord, returned LPDWord) error {
	return toError(Long(C.SCardControl(
		C.SCARDHANDLE(card),
		C.DWORD(controlCode),
		C.LPCVOID(unsafe.Pointer(send)),
		C.DWORD(sendLen),
		C.LPVOID(unsafe.Pointer(recv)),
		C.DWORD(recvLen),
		C.LPDWORD(unsafe.Pointer(returned)),
	)))
}

func (cgoBackend) Transmit(card SCardHandle, sendPci LPCSCardIORequest, send LPCByte, sendLen DWord, recvPci LPSCardIORequest, recv LPByte, recvLen LPDWord) error {
	return toError(Long(C.SCardTransmit(
		C.SCARDHANDLE(card),
		(*C.SCARD_IO_REQUEST)(unsafe.Pointer(sendPci)),
		C.LPCBYTE(unsafe.Pointer(send)),
		C.DWORD(sendLen),
		(*C.SCARD_IO_REQUEST)(unsafe.Pointer(recvPci)),
		C.LPBYTE(unsafe.Pointer(recv)),
		C.LPDWORD(unsafe.Pointer(recvLen)),
	)))
}

func (cgoBackend) ListReaderGroups(ctx SCardContext, groups LPStr, groupsLen LPDWord) error {
	return toError(Long(C.SCardListReaderGroups(
		C.SCARDCONTEXT(ctx),
		C.LPSTR(unsafe.Pointer(groups)),
		C.LPDWORD(unsafe.Pointer(groupsLen)),
	)))
}

func (cgoBackend) ListReaders(ctx SCardContext, groups LPCStr, readers LPStr, readersLen LPDWord) error {
	return toError(Long(C.SCardListReaders(
		C.SCARDCONTEXT(ctx),
		C.LPCSTR(unsafe.Pointer(groups)),
		C.LPSTR(unsafe.Pointer(readers)),
		C.LPDWORD(unsafe.Pointer(readersLen)),
	)))
}

func (cgoBackend) FreeMemory(ctx SCardContext, mem unsafe.Pointer) error {
	return toError(Long(C.SCardFreeMemory(C.SCARDCONTEXT(ctx), C.LPCVOID(mem))))
}

func (cgoBackend) Cancel(ctx SCardContext) error {
	return toError(Long(C.SCardCancel(C.SCARDCONTEXT(ctx))))
}

func (cgoBackend) GetAttrib(card SCardHandle, attrID DWord, attr LPByte, attrLen LPDWord) error {
	return toError(Long(C.SCardGetAttrib(
		C.SCARDHANDLE(card),
		C.DWORD(attrID),
		C.LPBYTE(unsafe.Pointer(attr)),
		C.LPDWORD(unsafe.Pointer(attrLen)),
	)))
}

func (cgoBackend) SetAttrib(card SCardHandle, attrID DWord, attr LPCByte, attrLen DWord) error {
	return toError(Long(C.SCardSetAttrib(
		C.SCARDHANDLE(card),
		C.DWORD(attrID),
		C.LPCBYTE(unsafe.Pointer(attr)),
		C.DWORD(attrLen),
	)))
}

func (cgoBackend) PCIT0() *SCardIORequest {
	return (*SCardIORequest)(unsafe.Pointer(&C.g_rgSCardT0Pci))
}

func (cgoBackend) PCIT1() *SCardIORequest {
	return (*SCardIORequest)(unsafe.Pointer(&C.g_rgSCardT1Pci))
}

func (cgoBackend) PCIRaw() *SCardIORequest {
	return (*SCardIORequest)(unsafe.Pointer(&C.g_rgSCardRawPci))
}

func (cgoBackend) StringifyError(rc Long) string {
	return C.GoString(C.pcsc_stringify_error(C.LONG(rc)))
}

func (cgoBackend) LibraryPath() string {
	return "" // CGO backends use system linker, no explicit path
}

func init() {
	backend = cgoBackend{}
}
