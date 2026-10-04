// SPDX-License-Identifier: BSD-3-Clause
package pcscgo

import "unsafe"

type (
	Byte   = byte
	UChar  = byte
	PUChar = *UChar
	UShort = uint16
)

type LPVoid = unsafe.Pointer

type (
	DWord  = ULong
	PDWord = *DWord
)

type (
	LPCStr  = *Byte
	LPCByte = *Byte
	LPByte  = *Byte
	LPDWord = *DWord
	LPStr   = *Byte
)

// these types were deprecated but still used by old drivers and applications. So just declare and use them.
type (
	LPTStr  = LPStr
	LPCTStr = LPCStr
)

// types unused by pcsc-lite
type (
	Bool   = int16
	Word   = uint16
	PULong = *ULong
)
