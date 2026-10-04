// SPDX-License-Identifier: BSD-3-Clause
//go:build (pcsc_static || pcsc_dynamic) && cgo

package pcscgo

import (
	"testing"
)

func TestBackendInterface_CGO(t *testing.T) {
	b := getBackend()
	if b == nil {
		t.Fatal("backend is nil")
	}

	// For CGO backends, LibraryPath should return empty string
	path := b.LibraryPath()
	if path != "" {
		t.Errorf("LibraryPath() = %q, want empty string for CGO backend", path)
	}

	// Test StringifyError
	errStr := b.StringifyError(0)
	t.Logf("StringifyError(0) = %q", errStr)

	// Test PCI functions
	pci0 := b.PCIT0()
	if pci0 == nil {
		t.Fatal("PCIT0() returned nil")
	}
	t.Logf("PCIT0().Protocol = %v", pci0.Protocol)

	pci1 := b.PCIT1()
	if pci1 == nil {
		t.Fatal("PCIT1() returned nil")
	}
	t.Logf("PCIT1().Protocol = %v", pci1.Protocol)

	pciRaw := b.PCIRaw()
	if pciRaw == nil {
		t.Fatal("PCIRaw() returned nil")
	}
	t.Logf("PCIRaw().Protocol = %v", pciRaw.Protocol)
}
