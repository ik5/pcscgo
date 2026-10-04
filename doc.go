// SPDX-License-Identifier: BSD-3-Clause
// Package pcscgo implement the pcsc-lite c library.
//
// The structs from PCSCTLVStructure through PINPropertiesStructure are declared
// in reader.h between "#pragma pack(push, 1)" and "#pragma pack(pop)", so the C
// compiler lays them out with no padding between fields.
//
// Go has no packing attribute. It aligns uint16 to 2 bytes and uint32 to 4, so
// the in-memory layout of these Go structs differs from the C wire layout.
// For example, PCSCTLVStructure is 6 bytes in C (Value at offset 2) but 8 bytes
// in Go (Value at offset 4).
//
// Because of this:
//   - Never cast a byte buffer to these types with unsafe, and never pass a
//     pointer to them to C. The layout does not match.
//   - Do not use unsafe.Sizeof or unsafe.Offsetof on them. They report the
//     padded Go layout. Use binary.Size for the C sizeof.
//   - Convert with encoding/binary (binary.Read / binary.Write), which encodes
//     fixed-size fields back to back with no padding.
//
// Byte order: PCSCTLVStructure.Value is always big-endian (PC/SC part 10).
// The PIN structures use host byte order.
package pcscgo
