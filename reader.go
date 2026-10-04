// SPDX-License-Identifier: BSD-3-Clause
package pcscgo

// Tags for requesting card and reader attributes
const (
	SCardClassVendorInfo     ULong = 1      // Vendor information definitions
	SCardClassCommunications ULong = 2      // Communication definitions
	SCardClassProtocol       ULong = 3      // Protocol definitions
	SCardClassPowerMgmt      ULong = 4      // Power Management definitions
	SCardClassSecurity       ULong = 5      // Security Assurance definitions
	SCardClassMechanical     ULong = 6      // Mechanical characteristic definitions
	SCardClassVendorDefined  ULong = 7      // Vendor specific definitions
	SCardClassIFDProtocol    ULong = 8      // Interface Device Protocol options
	SCardClassICCState       ULong = 9      // ICC State specific definitions
	SCardClassSystem         ULong = 0x7fff // System-specific definitions
)

const (
	SCardAttrVendorName           ULong = SCardClassVendorInfo<<16 | 0x0100     // Vendor name.
	SCardAttrVendorIFDType        ULong = SCardClassVendorInfo<<16 | 0x0101     // Vendor-supplied interface device type (model designation of reader).
	SCardAttrVendorIFDVersion     ULong = SCardClassVendorInfo<<16 | 0x0102     // Vendor-supplied interface device version (DWORD in the form 0xMMmmbbbb where MM = major version, mm = minor version, and bbbb = build number).
	SCardAttrVendorIFDSerialNo    ULong = SCardClassVendorInfo<<16 | 0x0103     // Vendor-supplied interface device serial number.
	SCardAttrChannelID            ULong = SCardClassCommunications<<16 | 0x0110 // DWORD encoded as 0xDDDDCCCC, where DDDD = data channel type and CCCC = channel number
	SCardAttrAsyncProtocolTypes   ULong = SCardClassProtocol<<16 | 0x0120
	SCardAttrDefaultClk           ULong = SCardClassProtocol<<16 | 0x0121 // Default clock rate, in kHz.
	SCardAttrMaxClk               ULong = SCardClassProtocol<<16 | 0x0122 // Maximum clock rate, in kHz.
	SCardAttrDefaultDataRate      ULong = SCardClassProtocol<<16 | 0x0123 // Default data rate, in bps.
	SCardAttrMaxDataRate          ULong = SCardClassProtocol<<16 | 0x0124 // Maximum data rate, in bps.
	SCardAttrMaxIFSD              ULong = SCardClassProtocol<<16 | 0x0125 // Maximum bytes for information file size device.
	SCardAttrSyncProtocolTypes    ULong = SCardClassProtocol<<16 | 0x0126
	SCardAttrPowerMgmtSupport     ULong = SCardClassPowerMgmt<<16 | 0x0131 // Zero if device does not support power down while smart card is inserted. Nonzero otherwise.
	SCardAttrUserToCardAuthDevice ULong = SCardClassSecurity<<16 | 0x0140
	SCardAttrUserAuthInputDevice  ULong = SCardClassSecurity<<16 | 0x0142
	SCardAttrCharacteristics      ULong = SCardClassMechanical<<16 | 0x0150 // DWORD indicating which mechanical characteristics are supported. If zero, no special characteristics are supported. Note that multiple bits can be set
	SCardAttrCurrentProtocolType  ULong = SCardClassIFDProtocol<<16 | 0x0201
	SCardAttrCurrentClk           ULong = SCardClassIFDProtocol<<16 | 0x0202 // Current clock rate, in kHz.
	SCardAttrCurrentF             ULong = SCardClassIFDProtocol<<16 | 0x0203 // Clock conversion factor.
	SCardAttrCurrentD             ULong = SCardClassIFDProtocol<<16 | 0x0204 // Bit rate conversion factor.
	SCardAttrCurrentN             ULong = SCardClassIFDProtocol<<16 | 0x0205 // Current guard time.
	SCardAttrCurrentW             ULong = SCardClassIFDProtocol<<16 | 0x0206 // Current work waiting time.
	SCardAttrCurrentIFSC          ULong = SCardClassIFDProtocol<<16 | 0x0207 // Current byte size for information field size card.
	SCardAttrCurrentIFSD          ULong = SCardClassIFDProtocol<<16 | 0x0208 // Current byte size for information field size device.
	SCardAttrCurrentBWT           ULong = SCardClassIFDProtocol<<16 | 0x0209 // Current block waiting time.
	SCardAttrCurrentCWT           ULong = SCardClassIFDProtocol<<16 | 0x020a // Current character waiting time.
	SCardAttrCurrentEBCEncoding   ULong = SCardClassIFDProtocol<<16 | 0x020b // Current error block control encoding.
	SCardAttrExtendedBWT          ULong = SCardClassIFDProtocol<<16 | 0x020c
	SCardAttrICCPresence          ULong = SCardClassICCState<<16 | 0x0300 // Single byte indicating smart card presence
	SCardAttrICCInterfaceStatus   ULong = SCardClassICCState<<16 | 0x0301 // Single byte. Zero if smart card electrical contact is not active; nonzero if contact is active.
	SCardAttrCurrentIOState       ULong = SCardClassICCState<<16 | 0x0302
	SCardAttrATRString            ULong = SCardClassICCState<<16 | 0x0303 // Answer to reset (ATR) string.
	SCardAttrICCTypePerATR        ULong = SCardClassICCState<<16 | 0x0304 // Single byte indicating smart card type
	SCardAttrESCReset             ULong = SCardClassVendorDefined<<16 | 0xA000
	SCardAttrESCCancel            ULong = SCardClassVendorDefined<<16 | 0xA003
	SCardAttrESCAuthrequest       ULong = SCardClassVendorDefined<<16 | 0xA005
	SCardAttrMaxinput             ULong = SCardClassVendorDefined<<16 | 0xA007
	SCardAttrDeviceUnit           ULong = SCardClassSystem<<16 | 0x0001 // Instance of this vendor's reader attached to the computer. The first instance will be device unit 0, the next will be unit 1 (if it is the same brand of reader) and so on. Two different brands of readers will both have zero for this value.
	SCardAttrDeviceInUse          ULong = SCardClassSystem<<16 | 0x0002 // Reserved for future use.
	SCardAttrDeviceFriendlyNameA  ULong = SCardClassSystem<<16 | 0x0003
	SCardAttrDeviceSystemNameA    ULong = SCardClassSystem<<16 | 0x0004
	SCardAttrDeviceFriendlyNameW  ULong = SCardClassSystem<<16 | 0x0005
	SCardAttrDeviceSystemNameW    ULong = SCardClassSystem<<16 | 0x0006
	SCardAttrSupressT1IFSRequest  ULong = SCardClassSystem<<16 | 0x0007
	SCardAttrDeviceFriendlyName         = SCardAttrDeviceFriendlyNameA // Reader's display name.
	SCardAttrDeviceSystemName           = SCardAttrDeviceSystemNameA   // Reader's system name.
)

const (
	CMIOCTLGetFeatureRequest ULong = 0x42000000 + 3400 // PC/SC part 10 v2.02.07 March 2010 reader tags
)

const (
	FeatureVerifyPINStart       Byte = 0x01
	FeatureVerifyPINFinish      Byte = 0x02
	FeatureModifyPINStart       Byte = 0x03
	FeatureModifyPINFinish      Byte = 0x04
	FeatureGetKeyPressed        Byte = 0x05
	FeatureVerifyPINDirect      Byte = 0x06 // Verify PIN
	FeatureModifyPINDirect      Byte = 0x07 // Modify PIN
	FeatureMCTReaderDirect      Byte = 0x08
	FeatureMCTUniversal         Byte = 0x09
	FeatureIFDPINProperties     Byte = 0x0A // retrieve properties of the IFD regarding PIN handling
	FeatureAbort                Byte = 0x0B
	FeatureSetSPEMessage        Byte = 0x0C
	FeatureVerifyPINDirectAppID Byte = 0x0D
	FeatureModifyPINDirectAppID Byte = 0x0E
	FeatureWriteDisplay         Byte = 0x0F
	FeatureGetKey               Byte = 0x10
	FeatureIFDDisplayProperties Byte = 0x11
	FeatureGetTLVProperties     Byte = 0x12
	FeatureCCIDESCCommand       Byte = 0x13
	FeatureExecutePACE          Byte = 0x20
)

// PCSCTLVStructure must be 6-bytes long
type PCSCTLVStructure struct {
	Tag    uint8
	Length uint8
	Value  uint32 // This value is always in BIG ENDIAN format as documented in PCSC v2 part 10 ch 2.2 page 2. You can use binary.BigEndian for example
}

// PINVerifyStructure is used with FeatureVerifyPINDirect.
//
// The C struct ends with a flexible array member (abData[]); the payload
// follows this header in the wire buffer and is not a field here.
type PINVerifyStructure struct {
	TimerOut                 uint8    // timeout is seconds (00 means use default timeout)
	TimerOut2                uint8    // timeout in seconds after first key stroke
	FormatString             uint8    // formatting options
	PINBlockString           uint8    // bits 7-4 bit size of PIN length in APDU, bits 3-0 PIN block size in bytes after justification and formatting
	PINLengthFormat          uint8    // bits 7-5 RFU, bit 4 set if system units are bytes, clear if system units are bits, bits 3-0 PIN length position in system units
	PINMaxExtraDigit         uint16   // 0xXXYY where XX is minimum PIN size in digits, and YY is maximum PIN size in digits
	EntryValidationCondition uint8    // Conditions under which PIN entry should be considered complete
	NumberMessage            uint8    // Number of messages to display for PIN verification
	LangID                   uint16   // Language for messages. https://docs.microsoft.com/en-us/windows/win32/intl/language-identifier-constants-and-strings
	MsgIndex                 uint8    // Message index (should be 00)
	TeoPrologue              [3]uint8 // T=1 block prologue field to use (fill with 00)
	DataLength               uint32   // length of the payload that follows this header
}

// PINModifyStructure is used with FeatureModifyPINDirect
//
// The C struct ends with a flexible array member (abData[]); the payload
// follows this header in the wire buffer and is not a field here.
type PINModifyStructure struct {
	TimerOut                 uint8    // timeout is seconds (00 means use default timeout)
	TimerOut2                uint8    // timeout in seconds after first key stroke
	FormatString             uint8    // formatting options
	PINBlockString           uint8    // bits 7-4 bit size of PIN length in APDU, bits 3-0 PIN block size in bytes after justification and formatting
	PINLengthFormat          uint8    // bits 7-5 RFU, bit 4 set if system units are bytes, clear if system units are bits, bits 3-0 PIN length position in system units
	InsertionOffsetOld       uint8    // Insertion position offset in bytes for the current PIN
	InsertionOffsetNew       uint8    // Insertion position offset in bytes for the new PIN
	PINMaxExtraDigit         uint16   // 0xXXYY where XX is minimum PIN size in digits, and YY is maximum PIN size in digits
	ConfirmPIN               uint8    // Flags governing need for confirmation of new PIN
	EntryValidationCondition uint8    // Conditions under which PIN entry should be considered complete
	NumberMessage            uint8    // Number of messages to display for PIN verification
	LangID                   uint16   // Language for messages. https://docs.microsoft.com/en-us/windows/win32/intl/language-identifier-constants-and-strings
	MsgIndex1                uint8    // index of 1st prompting message
	MsgIndex2                uint8    // index of 2nd prompting message
	MsgIndex3                uint8    // index of 3rd prompting message
	TeoPrologue              [3]uint8 // T=1 block prologue field to use (fill with 00)
	DataLength               uint32   // length of the payload that follows this header
}

// PINPropertiesStructure is used with FeatureIFDPINProperties
type PINPropertiesStructure struct {
	LCDLayout                uint16 // display characteristics
	EntryValidationCondition uint8
	TimeOut2                 uint8
}

// properties returned by FeatureGetTLVProperties
const (
	PCSCv2Part10PropertyLCDLayout                Byte = 1
	PCSCv2Part10PropertyEntryValidationCondition Byte = 2
	PCSCv2Part10PropertyTimeOut2                 Byte = 3
	PCSCv2Part10PropertyLCDMaxCharacters         Byte = 4
	PCSCv2Part10PropertyLCDMaxLines              Byte = 5
	PCSCv2Part10PropertyMinPINSize               Byte = 6
	PCSCv2Part10PropertyMaxPINSize               Byte = 7
	PCSCv2Part10PropertyFirmwareID               Byte = 8
	PCSCv2Part10PropertyPPDUSupport              Byte = 9
	PCSCv2Part10PropertyMaxAPDUDataSize          Byte = 10
	PCSCv2Part10PropertyIDVendor                 Byte = 11
	PCSCv2Part10PropertyIDProduct                Byte = 12
)

func SCardAttrValue(class, tag ULong) ULong {
	return (class << 16) | tag
}

// SCardCtlCode provides source compatibility on different platforms
func SCardCtlCode(code ULong) ULong {
	return 0x42000000 + code
}
