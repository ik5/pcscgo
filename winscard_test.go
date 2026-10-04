// SPDX-License-Identifier: BSD-3-Clause
package pcscgo

import (
	"errors"
	"testing"
)

func setupMockBackend() *mockBackend {
	m := &mockBackend{}
	SetBackend(m)
	return m
}

func TestSCardEstablishContext(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	var ctx SCardContext

	// Test success
	err := SCardEstablishContext(SCardScopeUser, nil, nil, &ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.EstablishContextCalls != 1 {
		t.Errorf("EstablishContextCalls = %d, want 1", m.EstablishContextCalls)
	}
	if m.LastEstablishContextScope != SCardScopeUser {
		t.Errorf("LastEstablishContextScope = %v, want %v", m.LastEstablishContextScope, SCardScopeUser)
	}

	// Test error from backend
	m.Reset()
	m.EstablishContextErr = &PCSCError{Code: SCardErrorNoService, Message: "Smart card service not available"}
	err = SCardEstablishContext(SCardScopeUser, nil, nil, &ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	var pcscErr *PCSCError
	if !errors.As(err, &pcscErr) {
		t.Fatalf("expected PCSCError, got %T", err)
	}
	if pcscErr.Code != SCardErrorNoService {
		t.Errorf("error code = %v, want %v", pcscErr.Code, SCardErrorNoService)
	}
}

func TestSCardReleaseContext(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardReleaseContext(SCardContext(123))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ReleaseContextCalls != 1 {
		t.Errorf("ReleaseContextCalls = %d, want 1", m.ReleaseContextCalls)
	}
}

func TestSCardIsValidContext(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardIsValidContext(SCardContext(123))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.IsValidContextCalls != 1 {
		t.Errorf("IsValidContextCalls = %d, want 1", m.IsValidContextCalls)
	}
}

func TestSCardConnect(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	var card SCardHandle
	var proto DWord

	// Test success
	err := SCardConnect(SCardContext(1), "Test Reader", SCardShareShared, SCardProtocolAny, &card, &proto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ConnectCalls != 1 {
		t.Errorf("ConnectCalls = %d, want 1", m.ConnectCalls)
	}
	if m.LastConnectReader == nil {
		t.Error("LastConnectReader should not be nil")
	}
	if m.LastConnectShareMode != SCardShareShared {
		t.Errorf("LastConnectShareMode = %v, want %v", m.LastConnectShareMode, SCardShareShared)
	}
	if m.LastConnectProtocols != SCardProtocolAny {
		t.Errorf("LastConnectProtocols = %v, want %v", m.LastConnectProtocols, SCardProtocolAny)
	}

	// Test embedded NUL in reader name
	m.Reset()
	err = SCardConnect(SCardContext(1), "Reader\x00NUL", SCardShareShared, SCardProtocolAny, &card, &proto)
	if err == nil {
		t.Fatal("expected error for embedded NUL")
	}
	if !errors.Is(err, SCardErrorInvalidParameter) {
		t.Errorf("expected SCardErrorInvalidParameter, got %v", err)
	}
}

func TestSCardReconnect(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	var proto DWord
	err := SCardReconnect(SCardHandle(1), SCardShareExclusive, SCardProtocolT1, SCardResetCard, &proto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ReconnectCalls != 1 {
		t.Errorf("ReconnectCalls = %d, want 1", m.ReconnectCalls)
	}
}

func TestSCardDisconnect(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardDisconnect(SCardHandle(1), SCardUnpowerCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.DisconnectCalls != 1 {
		t.Errorf("DisconnectCalls = %d, want 1", m.DisconnectCalls)
	}
	if m.LastDisconnectDisposition != SCardUnpowerCard {
		t.Errorf("LastDisconnectDisposition = %v, want %v", m.LastDisconnectDisposition, SCardUnpowerCard)
	}
}

func TestSCardBeginTransaction(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardBeginTransaction(SCardHandle(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.BeginTransactionCalls != 1 {
		t.Errorf("BeginTransactionCalls = %d, want 1", m.BeginTransactionCalls)
	}
}

func TestSCardEndTransaction(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardEndTransaction(SCardHandle(1), SCardResetCard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.EndTransactionCalls != 1 {
		t.Errorf("EndTransactionCalls = %d, want 1", m.EndTransactionCalls)
	}
}

func TestSCardStatus(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	readerName := make([]byte, 256)
	atr := make([]byte, MaxATRSize)

	readerLen, state, protocol, atrLen, err := SCardStatus(SCardHandle(1), readerName, atr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.StatusCalls != 1 {
		t.Errorf("StatusCalls = %d, want 1", m.StatusCalls)
	}
	_ = readerLen
	_ = state
	_ = protocol
	_ = atrLen
}

func TestSCardGetStatusChange(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	states := []SCardReaderState{
		{CurrentState: SCardStateUnaware},
		{CurrentState: SCardStateEmpty},
	}

	err := SCardGetStatusChange(SCardContext(1), Infinite, states)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.GetStatusChangeCalls != 1 {
		t.Errorf("GetStatusChangeCalls = %d, want 1", m.GetStatusChangeCalls)
	}
}

func TestSCardControl(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	send := []byte{0x01, 0x02, 0x03}
	recv := make([]byte, 256)

	n, err := SCardControl(SCardHandle(1), SCardCtlCode(3400), send, recv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ControlCalls != 1 {
		t.Errorf("ControlCalls = %d, want 1", m.ControlCalls)
	}
	if m.LastControlCode != SCardCtlCode(3400) {
		t.Errorf("LastControlCode = %v, want %v", m.LastControlCode, SCardCtlCode(3400))
	}
	if len(m.LastControlSend) != len(send) {
		t.Errorf("LastControlSend length = %d, want %d", len(m.LastControlSend), len(send))
	}
	_ = n
}

func TestSCardTransmit(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	send := []byte{0x00, 0xA4, 0x04, 0x00, 0x00}
	recv := make([]byte, 256)
	sendPci := SCardPCIT0()

	n, err := SCardTransmit(SCardHandle(1), sendPci, send, nil, recv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.TransmitCalls != 1 {
		t.Errorf("TransmitCalls = %d, want 1", m.TransmitCalls)
	}
	if m.LastTransmitSendPci != sendPci {
		t.Error("LastTransmitSendPci should match sendPci")
	}
	if len(m.LastTransmitSend) != len(send) {
		t.Errorf("LastTransmitSend length = %d, want %d", len(m.LastTransmitSend), len(send))
	}
	_ = n
}

func TestSCardListReaderGroups(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	groups := make([]byte, 256)
	n, err := SCardListReaderGroups(SCardContext(1), groups)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ListReaderGroupsCalls != 1 {
		t.Errorf("ListReaderGroupsCalls = %d, want 1", m.ListReaderGroupsCalls)
	}
	_ = n
}

func TestSCardListReaders(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	readers := make([]byte, 256)
	n, err := SCardListReaders(SCardContext(1), "", readers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ListReadersCalls != 1 {
		t.Errorf("ListReadersCalls = %d, want 1", m.ListReadersCalls)
	}
	_ = n
}

func TestSCardFreeMemory(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardFreeMemory(SCardContext(1), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.FreeMemoryCalls != 1 {
		t.Errorf("FreeMemoryCalls = %d, want 1", m.FreeMemoryCalls)
	}
}

func TestSCardCancel(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	err := SCardCancel(SCardContext(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.CancelCalls != 1 {
		t.Errorf("CancelCalls = %d, want 1", m.CancelCalls)
	}
}

func TestSCardGetAttrib(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	attr := make([]byte, 256)
	n, err := SCardGetAttrib(SCardHandle(1), SCardAttrVendorName, attr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.GetAttribCalls != 1 {
		t.Errorf("GetAttribCalls = %d, want 1", m.GetAttribCalls)
	}
	if m.LastGetAttribID != SCardAttrVendorName {
		t.Errorf("LastGetAttribID = %v, want %v", m.LastGetAttribID, SCardAttrVendorName)
	}
	_ = n
}

func TestSCardSetAttrib(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	attr := []byte{0x01, 0x02}
	err := SCardSetAttrib(SCardHandle(1), SCardAttrDeviceFriendlyName, attr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SetAttribCalls != 1 {
		t.Errorf("SetAttribCalls = %d, want 1", m.SetAttribCalls)
	}
	if m.LastSetAttribID != SCardAttrDeviceFriendlyName {
		t.Errorf("LastSetAttribID = %v, want %v", m.LastSetAttribID, SCardAttrDeviceFriendlyName)
	}
}

func TestSCardPCIT0(t *testing.T) {
	_ = setupMockBackend()
	defer SetBackend(nil)

	pci := SCardPCIT0()
	if pci == nil {
		t.Fatal("PCIT0() returned nil")
	}
	if pci.Protocol != SCardProtocolT0 {
		t.Errorf("Protocol = %v, want %v", pci.Protocol, SCardProtocolT0)
	}
}

func TestSCardPCIT1(t *testing.T) {
	_ = setupMockBackend()
	defer SetBackend(nil)

	pci := SCardPCIT1()
	if pci == nil {
		t.Fatal("PCIT1() returned nil")
	}
	if pci.Protocol != SCardProtocolT1 {
		t.Errorf("Protocol = %v, want %v", pci.Protocol, SCardProtocolT1)
	}
}

func TestSCardPCIRaw(t *testing.T) {
	_ = setupMockBackend()
	defer SetBackend(nil)

	pci := SCardPCIRaw()
	if pci == nil {
		t.Fatal("PCIRaw() returned nil")
	}
	if pci.Protocol != SCardProtocolRaw {
		t.Errorf("Protocol = %v, want %v", pci.Protocol, SCardProtocolRaw)
	}
}

func TestPCSCStringifyError(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	m.StringifyErrorResult = "Mock error message"
	err := PCSCStringifyError(SCardErrorTimeout)
	if err != "Mock error message" {
		t.Errorf("PCSCStringifyError = %q, want %q", err, "Mock error message")
	}
}

func TestPCSCLibraryPath_Mock(t *testing.T) {
	_ = setupMockBackend()
	defer SetBackend(nil)

	path := PCSCLibraryPath()
	if path != "/mock/libpcsclite.so" {
		t.Errorf("PCSCLibraryPath = %q, want %q", path, "/mock/libpcsclite.so")
	}
}

func TestErrorPropagation(t *testing.T) {
	m := setupMockBackend()
	defer SetBackend(nil)

	var card SCardHandle
	var proto DWord

	m.ConnectErr = &PCSCError{Code: SCardErrorNoReadersAvailable, Message: "No readers available"}
	err := SCardConnect(SCardContext(1), "Test Reader", SCardShareShared, SCardProtocolAny, &card, &proto)
	if err == nil {
		t.Fatal("expected error")
	}

	var pcscErr *PCSCError
	if !errors.As(err, &pcscErr) {
		t.Fatalf("expected PCSCError, got %T", err)
	}
	if pcscErr.Code != SCardErrorNoReadersAvailable {
		t.Errorf("error code = %v, want %v", pcscErr.Code, SCardErrorNoReadersAvailable)
	}
	if pcscErr.Message == "" {
		t.Error("error message should not be empty")
	}
}
