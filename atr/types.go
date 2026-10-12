package atr

// InitialCharacter first bits to explain how to read the rest of the ATR.
type InitialCharacter uint8

// TAi, TBi, TCi, TDi datatype allowing to bind exact needed
type (
	TA uint8
	TB uint8
	TC uint8
	TD uint8
)

type Group struct {
	// TA - clock and speed
	TA *TA

	// TB - programming voltage (VPP) - Obsolete for modern cards/
	TB *TB

	// TC - extra guard time.
	TC *TC

	// TD - protocol announcements
	TD *TD

	// Protocol - low nibble of this group's TD
	Protocol uint8

	// Index shows the position of TXi in the list starting in 1 because TA..TD	 always starts with 1 and ending in i
	Index uint
}

type ATR struct {
	// TS - Initial Character
	TS InitialCharacter

	// T0 - the table of contents
	T0 uint8

	// Groups - parameter group, length by T0 field calculation for number of fields
	Groups []Group

	// Historical -  the free-form ID card
	Historical []byte

	// TCK - checksum only if there at least one TDi in groups.
	TCK *uint8
}

const (
	// direct convention: bits are sent as-is. This is the usual TS.
	DirectConvention InitialCharacter = 0x3B

	// inverse convention: every byte is transformed (see ,,/smart_card_basics.md#what-inverse-actually-means).
	InverseConvention InitialCharacter = 0x3F
)
