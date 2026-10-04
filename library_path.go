// SPDX-License-Identifier: BSD-3-Clause
//go:build !pcsc_purego

package pcscgo

import "os"

// defaultLibraryPath returns the default library path for the current platform.
// For CGO backends, this returns an empty string since the library is linked at compile time.
// It can be overridden by setting the PCSCLITE_LIB_PATH environment variable (purego only).
func defaultLibraryPath() string {
	if v := os.Getenv("PCSCLITE_LIB_PATH"); v != "" {
		return v
	}
	// CGO backends don't use dynamic loading, return empty
	return ""
}
