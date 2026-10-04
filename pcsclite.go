// SPDX-License-Identifier: BSD-3-Clause
package pcscgo

import (
	"fmt"
	"unsafe"
)

// hContext returned by SCardEstablishContext()
type (
	SCardContext   = Long
	PSCardContext  = *SCardContext
	LPSCARDCONTEXT = *SCardContext
)

// hCard returned by SCardConnect()
type (
	SCardHandle   = Long
	PSCardHandle  = *SCardHandle
	LPScardHandle = *SCardHandle
)

// MaxATRSize maximum size for the ATR
const MaxATRSize = 33

// SCardReaderState mirrors SCARD_READERSTATE
type SCardReaderState struct {
	Reader       *Byte
	UserData     unsafe.Pointer
	CurrentState DWord
	EventState   DWord
	ATRLength    DWord            // cbAtr: number of valid bytes in ATR up to MaxATRSize
	ATR          [MaxATRSize]Byte // rgbAtr: ATR bytes, only the first ATRLength are valid
}

// LPSCardReaderState mirrors LPSCARD_READERSTATE
type LPSCardReaderState = *SCardReaderState

// SCardIORequest holds Protocol Control Information (PCI).
type SCardIORequest struct {
	Protocol  ULong // Protocol identifier
	PCILength ULong // Protocol Control Information Length
}

type (
	// PSCardIORequest mirrors PSCARD_IO_REQUEST.
	PSCardIORequest = *SCardIORequest

	// LPSCardIORequest mirrors LPSCARD_IO_REQUEST.
	LPSCardIORequest = *SCardIORequest

	// LPCSCardIORequest mirrors LPCSCARD_IO_REQUEST (const SCARD_IO_REQUEST *).
	// Go has no const pointers, so the callee must treat it as read-only.
	LPCSCardIORequest = *SCardIORequest
)

// ErrorCode is a PC/SC status code. It is always 32 bits wide.
//
// The header defines the codes as (LONG)0x8010xxxx. On 32-bit platforms
// LONG is int32, and Go rejects constants such as 0x80100001 as overflowing it,
// so the codes are typed uint32 instead. Convert a raw LONG from the library
// with ErrorCode(uint32(rc)).
type ErrorCode uint32

// CodeOf converts the raw LONG returned by the library into an ErrorCode.
// It keeps the low 32 bits, so the result is the same on 32-bit and 64-bit.
func CodeOf(rc Long) ErrorCode { return ErrorCode(uint32(rc)) }

// AsLong converts an ErrorCode to the LONG a C call would return.
func (errCode ErrorCode) AsLong() Long {
	return Long(errCode)
}

// Error implements the error interface for ErrorCode.
// Format: "PC/SC error 0x80100001"
func (errCode ErrorCode) Error() string {
	return fmt.Sprintf("PC/SC error %#x", uint32(errCode))
}

// Error codes returned by the PC/SC API (SCARD_* in pcsclite.h).
//
// Header prefix mapping: SCARD_S_ is SCardSuccess, SCARD_F_ is SCardFatal,
// SCARD_E_ is SCardError, SCARD_W_ is SCardWarning. SCARD_P_SHUTDOWN is
// SCardShutdown.
//
// The descriptions come from
// http://msdn.microsoft.com/en-us/library/aa924526.aspx
const (
	SCardSuccess                     ErrorCode = 0x00000000              // No error was encountered.
	SCardFatalInternalError          ErrorCode = 0x80100001              // An internal consistency check failed.
	SCardErrorCancelled              ErrorCode = 0x80100002              // The action was cancelled by an SCardCancel request.
	SCardErrorInvalidHandle          ErrorCode = 0x80100003              // The supplied handle was invalid.
	SCardErrorInvalidParameter       ErrorCode = 0x80100004              // One or more of the supplied parameters could not be properly interpreted.
	SCardErrorInvalidTarget          ErrorCode = 0x80100005              // Registry startup information is missing or invalid.
	SCardErrorNoMemory               ErrorCode = 0x80100006              // Not enough memory available to complete this command.
	SCardFatalWaitedTooLong          ErrorCode = 0x80100007              // An internal consistency timer has expired.
	SCardErrorInsufficientBuffer     ErrorCode = 0x80100008              // The data buffer to receive returned data is too small for the returned data.
	SCardErrorUnknownReader          ErrorCode = 0x80100009              // The specified reader name is not recognized.
	SCardErrorTimeout                ErrorCode = 0x8010000A              // The user-specified timeout value has expired.
	SCardErrorSharingViolation       ErrorCode = 0x8010000B              // The smart card cannot be accessed because of other connections outstanding.
	SCardErrorNoSmartcard            ErrorCode = 0x8010000C              // The operation requires a Smart Card, but no Smart Card is currently in the device.
	SCardErrorUnknownCard            ErrorCode = 0x8010000D              // The specified smart card name is not recognized.
	SCardErrorCantDispose            ErrorCode = 0x8010000E              // The system could not dispose of the media in the requested manner.
	SCardErrorProtoMismatch          ErrorCode = 0x8010000F              // The requested protocols are incompatible with the protocol currently in use with the smart card.
	SCardErrorNotReady               ErrorCode = 0x80100010              // The reader or smart card is not ready to accept commands.
	SCardErrorInvalidValue           ErrorCode = 0x80100011              // One or more of the supplied parameters values could not be properly interpreted.
	SCardErrorSystemCancelled        ErrorCode = 0x80100012              // The action was cancelled by the system, presumably to log off or shut down.
	SCardFatalCommError              ErrorCode = 0x80100013              // An internal communications error has been detected.
	SCardFatalUnknownError           ErrorCode = 0x80100014              // An internal error has been detected, but the source is unknown.
	SCardErrorInvalidATR             ErrorCode = 0x80100015              // An ATR obtained from the registry is not a valid ATR string.
	SCardErrorNotTransacted          ErrorCode = 0x80100016              // An attempt was made to end a non-existent transaction.
	SCardErrorReaderUnavailable      ErrorCode = 0x80100017              // The specified reader is not currently available for use.
	SCardShutdown                    ErrorCode = 0x80100018              // The operation has been aborted to allow the server application to exit.
	SCardErrorPCITooSmall            ErrorCode = 0x80100019              // The PCI Receive buffer was too small.
	SCardErrorReaderUnsupported      ErrorCode = 0x8010001A              // The reader driver does not meet minimal requirements for support.
	SCardErrorDuplicateReader        ErrorCode = 0x8010001B              // The reader driver did not produce a unique reader name.
	SCardErrorCardUnsupported        ErrorCode = 0x8010001C              // The smart card does not meet minimal requirements for support.
	SCardErrorNoService              ErrorCode = 0x8010001D              // The Smart card resource manager is not running.
	SCardErrorServiceStopped         ErrorCode = 0x8010001E              // The Smart card resource manager has shut down.
	SCardErrorUnexpected             ErrorCode = 0x8010001F              // An unexpected card error has occurred.
	SCardErrorUnsupportedFeature     ErrorCode = 0x8010001F              // This smart card does not support the requested feature.
	SCardErrorICCInstallation        ErrorCode = 0x80100020              // No primary provider can be found for the smart card.
	SCardErrorICCCreateorder         ErrorCode = 0x80100021              // The requested order of object creation is not supported.
	SCardErrorDirNotFound            ErrorCode = 0x80100023              // The identified directory does not exist in the smart card.
	SCardErrorFileNotFound           ErrorCode = 0x80100024              // The identified file does not exist in the smart card.
	SCardErrorNoDir                  ErrorCode = 0x80100025              // The supplied path does not represent a smart card directory.
	SCardErrorNoFile                 ErrorCode = 0x80100026              // The supplied path does not represent a smart card file.
	SCardErrorNoAccess               ErrorCode = 0x80100027              // Access is denied to this file.
	SCardErrorWriteTooMany           ErrorCode = 0x80100028              // The smart card does not have enough memory to store the information.
	SCardErrorBadSeek                ErrorCode = 0x80100029              // There was an error trying to set the smart card file object pointer.
	SCardErrorInvalidCHV             ErrorCode = 0x8010002A              // The supplied PIN is incorrect.
	SCardErrorUnknownResMsg          ErrorCode = 0x8010002B              // An unrecognized error code was returned from a layered component.
	SCardErrorUnknownResMng                    = SCardErrorUnknownResMsg // SCardErrorUnknownResMng is an alias of SCardErrorUnknownResMsg
	SCardErrorNoSuchCertificate      ErrorCode = 0x8010002C              // The requested certificate does not exist.
	SCardErrorCertificateUnavailable ErrorCode = 0x8010002D              // The requested certificate could not be obtained.
	SCardErrorNoReadersAvailable     ErrorCode = 0x8010002E              // Cannot find a smart card reader.
	SCardErrorCommDataLost           ErrorCode = 0x8010002F              // A communications error with the smart card has been detected. Retry the operation.
	SCardErrorNoKeyContainer         ErrorCode = 0x80100030              // The requested key container does not exist on the smart card.
	SCardErrorServerTooBusy          ErrorCode = 0x80100031              // The Smart Card Resource Manager is too busy to complete this operation.
	SCardWarningUnsupportedCard      ErrorCode = 0x80100065              // The reader cannot communicate with the card, due to ATR string configuration conflicts.
	SCardWarningUnresponsiveCard     ErrorCode = 0x80100066              // The smart card is not responding to a reset.
	SCardWarningUnpoweredCard        ErrorCode = 0x80100067              // Power has been removed from the smart card, so that further communication is not possible.
	SCardWarningResetCard            ErrorCode = 0x80100068              // The smart card has been reset, so any shared state information is invalid.
	SCardWarningRemovedCard          ErrorCode = 0x80100069              // The smart card has been removed, so further communication is not possible.
	SCardWarningSecurityViolation    ErrorCode = 0x8010006A              // Access was denied because of a security violation.
	SCardWarningWrongCHV             ErrorCode = 0x8010006B              // The card cannot be accessed because the wrong PIN was presented.
	SCardWarningCHVBlocked           ErrorCode = 0x8010006C              // The card cannot be accessed because the maximum number of PIN entry attempts has been reached.
	SCardWarningEOF                  ErrorCode = 0x8010006D              // The end of the smart card file has been reached.
	SCardWarningCancelledByUser      ErrorCode = 0x8010006E              // The user pressed "Cancel" on a Smart Card Selection Dialog.
	SCardWarningCardNotAuthenticated ErrorCode = 0x8010006F              // No PIN was presented to the smart card.

	// SCardErrorUnsupportedFeature is also commented out in the header.
	// SCardErrorUnsupportedFeature ErrorCode = 0x80100022 // This smart card does not support the requested feature.
)

// SCardAutoallocate mirrors SCARD_AUTOALLOCATE, ((DWORD)-1).
// See SCardFreeMemory
const SCardAutoallocate DWord = ^DWord(0)

// Scoping constants
const (
	SCardScopeUser     DWord = 0x0000 // Scope in user space
	SCardScopeTerminal DWord = 0x0001 // Scope in terminal
	SCardScopeSystem   DWord = 0x0002 // Scope in system
	SCardScopeGlobal   DWord = 0x0003 // Scope is global
)

// Protocol Identifiers
const (
	SCardProtocolUndefined DWord = 0x0000 // protocol not set
	SCardProtocolT0        DWord = 0x0001 // T=0 active protocol.
	SCardProtocolT1        DWord = 0x0002 // T=1 active protocol.
	SCardProtocolRaw       DWord = 0x0004 // Raw active protocol.
	SCardProtocolT15       DWord = 0x0008 // T=15 protocol.

	// SCardProtocolUnset is kept for backward compatibility with the header.
	SCardProtocolUnset = SCardProtocolUndefined

	// SCardProtocolAny: IFD determines protocol.
	SCardProtocolAny = SCardProtocolT0 | SCardProtocolT1
)

// Share modes for SCardConnect (SCARD_SHARE_*).
const (
	SCardShareExclusive DWord = 0x0001 // Exclusive mode only
	SCardShareShared    DWord = 0x0002 // Shared mode only
	SCardShareDirect    DWord = 0x0003 // Raw mode only
)

// Card dispositions for SCardDisconnect, SCardReconnect and SCardEndTransaction (SCARD_*_CARD).
const (
	SCardLeaveCard   DWord = 0x0000 // Do nothing on close
	SCardResetCard   DWord = 0x0001 // Reset on close
	SCardUnpowerCard DWord = 0x0002 // Power down on close
	SCardEjectCard   DWord = 0x0003 // Eject on close
)

// Card states returned by SCardStatus (SCARD_UNKNOWN ... SCARD_SPECIFIC).
const (
	SCardUnknown    DWord = 0x0001 // Unknown state
	SCardAbsent     DWord = 0x0002 // Card is absent
	SCardPresent    DWord = 0x0004 // Card is present
	SCardSwallowed  DWord = 0x0008 // Card not powered
	SCardPowered    DWord = 0x0010 // Card is powered
	SCardNegotiable DWord = 0x0020 // Ready for PTS
	SCardSpecific   DWord = 0x0040 // PTS has been set
)

// Reader state flags for SCARD_READERSTATE.CurrentState and EventState (SCARD_STATE_*), used with SCardGetStatusChange.
const (
	SCardStateUnaware     DWord = 0x0000 // App wants status
	SCardStateIgnore      DWord = 0x0001 // Ignore this reader
	SCardStateChanged     DWord = 0x0002 // State has changed
	SCardStateUnknown     DWord = 0x0004 // Reader unknown
	SCardStateUnavailable DWord = 0x0008 // Status unavailable
	SCardStateEmpty       DWord = 0x0010 // Card removed
	SCardStatePresent     DWord = 0x0020 // Card inserted
	SCardStateATRMatch    DWord = 0x0040 // ATR matches card
	SCardStateExclusive   DWord = 0x0080 // Exclusive Mode
	SCardStateInUse       DWord = 0x0100 // Shared Mode
	SCardStateMute        DWord = 0x0200 // Unresponsive card
	SCardStateUnpowered   DWord = 0x0400 // Unpowered card
)

// Infinite mirrors INFINITE: infinite timeout.
const Infinite DWord = 0xFFFFFFFF

// PCSCLiteVersionNumber mirrors PCSCLITE_VERSION_NUMBER: the pcsc-lite version of the header this binding was
// translated from. It is not the version of the library loaded at runtime.
const PCSCLiteVersionNumber = "2.5.2"

// PCSCLiteMaxReadersContexts mirrors PCSCLITE_MAX_READERS_CONTEXTS:
// maximum readers context (a slot is counted as a reader).
const PCSCLiteMaxReadersContexts = 16

// MaxReaderName mirrors MAX_READERNAME: maximum length of a reader name.
const MaxReaderName = 128

// SCardATRLength is the maximum ATR size
const SCardATRLength DWord = MaxATRSize

// MaxBufferSize mirrors MAX_BUFFER_SIZE: maximum Tx/Rx buffer for a short APDU.
const MaxBufferSize = 264

// MaxBufferSizeExtended mirrors MAX_BUFFER_SIZE_EXTENDED: enhanced
// (64K + APDU + Lc + Le + SW) Tx/Rx buffer.
const MaxBufferSizeExtended = 4 + 3 + (1 << 16) + 3 + 2
