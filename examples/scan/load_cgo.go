// SPDX-License-Identifier: BSD-3-Clause
//go:build !pcsc_purego

package main

import (
	"fmt"

	"github.com/ik5/pcscgo"
)

func loadLibrary(libPath string, verbose bool) error {
	if verbose {
		path := pcscgo.PCSCLibraryPath()
		if path != "" {
			fmt.Printf("Using library path: %s\n", path)
		} else {
			fmt.Println("Using CGO backend (library linked at compile time)")
		}
	}
	return nil
}
