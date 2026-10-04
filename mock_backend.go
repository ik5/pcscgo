package pcscgo

import (
	"sync"
	"unsafe"
)

// mockBackend is a mock implementation of the Backend interface for testing.
type mockBackend struct {
	mu sync.Mutex

	// Call tracking
	EstablishContextCalls int
	ReleaseContextCalls   int
	IsValidContextCalls   int
	ConnectCalls          int
	ReconnectCalls        int
	DisconnectCalls       int
	BeginTransactionCalls int
	EndTransactionCalls   int
	StatusCalls           int
	GetStatusChangeCalls  int
	ControlCalls          int
	TransmitCalls         int
	ListReaderGroupsCalls int
	ListReadersCalls      int
	FreeMemoryCalls       int
	CancelCalls           int
	GetAttribCalls        int
	SetAttribCalls        int

	// Return values / errors
	EstablishContextErr error
	ReleaseContextErr   error
	IsValidContextErr   error
	ConnectErr          error
	ReconnectErr        error
	DisconnectErr       error
	BeginTransactionErr error
	EndTransactionErr   error
	StatusErr           error
	GetStatusChangeErr  error
	ControlErr          error
	TransmitErr         error
	ListReaderGroupsErr error
	ListReadersErr      error
	FreeMemoryErr       error
	CancelErr           error
	GetAttribErr        error
	SetAttribErr        error

	// Capture last arguments for verification
	LastEstablishContextScope DWord
	LastConnectReader         LPCStr
	LastConnectShareMode      DWord
	LastConnectProtocols      DWord
	LastDisconnectDisposition DWord
	LastTransmitSendPci       LPCSCardIORequest
	LastTransmitSend          []byte
	LastTransmitRecvPci       LPSCardIORequest
	LastControlCode           DWord
	LastControlSend           []byte
	LastGetAttribID           DWord
	LastSetAttribID           DWord

	// PCI pointers to return
	PCIT0Ptr  *SCardIORequest
	PCIT1Ptr  *SCardIORequest
	PCIRawPtr *SCardIORequest

	// StringifyError return
	StringifyErrorResult string
}

func (m *mockBackend) EstablishContext(scope DWord, r1, r2 unsafe.Pointer, ctx *SCardContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EstablishContextCalls++
	m.LastEstablishContextScope = scope
	return m.EstablishContextErr
}

func (m *mockBackend) ReleaseContext(ctx SCardContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReleaseContextCalls++
	return m.ReleaseContextErr
}

func (m *mockBackend) IsValidContext(ctx SCardContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.IsValidContextCalls++
	return m.IsValidContextErr
}

func (m *mockBackend) Connect(ctx SCardContext, reader LPCStr, shareMode, preferredProtocols DWord, card *SCardHandle, activeProtocol LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalls++
	m.LastConnectReader = reader
	m.LastConnectShareMode = shareMode
	m.LastConnectProtocols = preferredProtocols
	return m.ConnectErr
}

func (m *mockBackend) Reconnect(card SCardHandle, shareMode, preferredProtocols, initialization DWord, activeProtocol LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReconnectCalls++
	return m.ReconnectErr
}

func (m *mockBackend) Disconnect(card SCardHandle, disposition DWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DisconnectCalls++
	m.LastDisconnectDisposition = disposition
	return m.DisconnectErr
}

func (m *mockBackend) BeginTransaction(card SCardHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BeginTransactionCalls++
	return m.BeginTransactionErr
}

func (m *mockBackend) EndTransaction(card SCardHandle, disposition DWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndTransactionCalls++
	return m.EndTransactionErr
}

func (m *mockBackend) Status(card SCardHandle, readerName LPStr, readerLen, state, protocol LPDWord, atr LPByte, atrLen LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusCalls++
	return m.StatusErr
}

func (m *mockBackend) GetStatusChange(ctx SCardContext, timeout DWord, states *SCardReaderState, n DWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetStatusChangeCalls++
	return m.GetStatusChangeErr
}

func (m *mockBackend) Control(card SCardHandle, controlCode DWord, send LPCByte, sendLen DWord, recv LPByte, recvLen DWord, returned LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ControlCalls++
	m.LastControlCode = controlCode
	if send != nil && sendLen > 0 {
		m.LastControlSend = unsafe.Slice(send, sendLen)
	}
	return m.ControlErr
}

func (m *mockBackend) Transmit(card SCardHandle, sendPci LPCSCardIORequest, send LPCByte, sendLen DWord, recvPci LPSCardIORequest, recv LPByte, recvLen LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TransmitCalls++
	m.LastTransmitSendPci = sendPci
	if send != nil && sendLen > 0 {
		m.LastTransmitSend = unsafe.Slice(send, sendLen)
	}
	m.LastTransmitRecvPci = recvPci
	return m.TransmitErr
}

func (m *mockBackend) ListReaderGroups(ctx SCardContext, groups LPStr, groupsLen LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListReaderGroupsCalls++
	return m.ListReaderGroupsErr
}

func (m *mockBackend) ListReaders(ctx SCardContext, groups LPCStr, readers LPStr, readersLen LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListReadersCalls++
	return m.ListReadersErr
}

func (m *mockBackend) FreeMemory(ctx SCardContext, mem unsafe.Pointer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FreeMemoryCalls++
	return m.FreeMemoryErr
}

func (m *mockBackend) Cancel(ctx SCardContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CancelCalls++
	return m.CancelErr
}

func (m *mockBackend) GetAttrib(card SCardHandle, attrID DWord, attr LPByte, attrLen LPDWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetAttribCalls++
	m.LastGetAttribID = attrID
	return m.GetAttribErr
}

func (m *mockBackend) SetAttrib(card SCardHandle, attrID DWord, attr LPCByte, attrLen DWord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SetAttribCalls++
	m.LastSetAttribID = attrID
	return m.SetAttribErr
}

func (m *mockBackend) PCIT0() *SCardIORequest {
	if m.PCIT0Ptr != nil {
		return m.PCIT0Ptr
	}
	return &SCardIORequest{Protocol: SCardProtocolT0}
}

func (m *mockBackend) PCIT1() *SCardIORequest {
	if m.PCIT1Ptr != nil {
		return m.PCIT1Ptr
	}
	return &SCardIORequest{Protocol: SCardProtocolT1}
}

func (m *mockBackend) PCIRaw() *SCardIORequest {
	if m.PCIRawPtr != nil {
		return m.PCIRawPtr
	}
	return &SCardIORequest{Protocol: SCardProtocolRaw}
}

func (m *mockBackend) StringifyError(rc Long) string {
	if m.StringifyErrorResult != "" {
		return m.StringifyErrorResult
	}
	return "mock error"
}

func (m *mockBackend) LibraryPath() string {
	return "/mock/libpcsclite.so"
}

func (m *mockBackend) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EstablishContextCalls = 0
	m.ReleaseContextCalls = 0
	m.IsValidContextCalls = 0
	m.ConnectCalls = 0
	m.ReconnectCalls = 0
	m.DisconnectCalls = 0
	m.BeginTransactionCalls = 0
	m.EndTransactionCalls = 0
	m.StatusCalls = 0
	m.GetStatusChangeCalls = 0
	m.ControlCalls = 0
	m.TransmitCalls = 0
	m.ListReaderGroupsCalls = 0
	m.ListReadersCalls = 0
	m.FreeMemoryCalls = 0
	m.CancelCalls = 0
	m.GetAttribCalls = 0
	m.SetAttribCalls = 0

	m.EstablishContextErr = nil
	m.ReleaseContextErr = nil
	m.IsValidContextErr = nil
	m.ConnectErr = nil
	m.ReconnectErr = nil
	m.DisconnectErr = nil
	m.BeginTransactionErr = nil
	m.EndTransactionErr = nil
	m.StatusErr = nil
	m.GetStatusChangeErr = nil
	m.ControlErr = nil
	m.TransmitErr = nil
	m.ListReaderGroupsErr = nil
	m.ListReadersErr = nil
	m.FreeMemoryErr = nil
	m.CancelErr = nil
	m.GetAttribErr = nil
	m.SetAttribErr = nil

	m.LastConnectReader = nil
	m.LastTransmitSend = nil
	m.LastTransmitSendPci = nil
	m.LastTransmitRecvPci = nil
	m.LastControlSend = nil
}
