// SPDX-License-Identifier: BSD-3-Clause
//go:build pcsc_purego

package main

import (
	"fmt"

	"github.com/ik5/pcscgo"
)

func loadLibrary(libPath string, verbose bool) error {
	if libPath != "" {
		if err := pcscgo.Load(libPath); err != nil {
			return fmt.Errorf("failed to load library: %w", err)
		}
		if verbose {
			fmt.Printf("Loaded library from: %s\n", libPath)
		}
	} else if verbose {
		path := pcscgo.PCSCLibraryPath()
		if path != "" {
			fmt.Printf("Using default library path: %s\n", path)
		}
	}
	return nil
}
