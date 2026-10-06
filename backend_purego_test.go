// SPDX-License-Identifier: BSD-3-Clause
//go:build pcsc_purego

package pcscgo

import (
	"errors"
	"os"
	"testing"
)

func TestPuregoLoad_InvalidPath(t *testing.T) {
	err := Load("/nonexistent/path/libpcsclite.so.1")
	if errors.Is(err, ErrAlreadyLoaded) {
		t.Skip("library already loaded by an earlier test; dlopen path not reachable")
	}
	if err == nil {
		t.Fatal("expected error for nonexistent library")
	}
	if !IsNotLoaded(err) {
		t.Errorf("IsNotLoaded() = false, want true for load error")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	if le.Op != "dlopen" {
		t.Errorf("LoadError.Op = %q, want %q", le.Op, "dlopen")
	}
}

func TestPuregoLoad_EnvVarOverride(t *testing.T) {
	// Save original env
	orig := os.Getenv("PCSCLITE_LIB_PATH")
	defer func() {
		if orig != "" {
			os.Setenv("PCSCLITE_LIB_PATH", orig)
		} else {
			os.Unsetenv("PCSCLITE_LIB_PATH")
		}
	}()

	// Test with custom path via env var (will fail to load but should use the path)
	os.Setenv("PCSCLITE_LIB_PATH", "/custom/path/libpcsclite.so")
	err := Load("") // Empty string should use defaultLibraryPath() which reads env
	// We expect an error since the path doesn't exist, but it should be a dlopen error
	if err == nil {
		t.Fatal("expected error for nonexistent library")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	// The operation should be dlopen with the custom path
	t.Logf("Load error: %v", err)
}

func TestPuregoLoad_AlreadyLoaded(t *testing.T) {
	// Save original env
	orig := os.Getenv("PCSCLITE_LIB_PATH")
	defer func() {
		if orig != "" {
			os.Setenv("PCSCLITE_LIB_PATH", orig)
		} else {
			os.Unsetenv("PCSCLITE_LIB_PATH")
		}
	}()

	// Use a path that might exist (won't actually load, but tests the logic)
	// We can't easily test "already loaded" without a real library, so test the error type
	err := Load("/definitely/does/not/exist.so")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotLoaded(err) {
		t.Error("IsNotLoaded should be true for failed load")
	}
}

func TestPuregoLoad_ExplicitThenUse(t *testing.T) {
	if err := Load("libpcsclite.so.1"); err != nil && !errors.Is(err, ErrAlreadyLoaded) {
		t.Skipf("libpcsclite not available: %v", err)
	}
	var ctx SCardContext
	err := SCardEstablishContext(SCardScopeUser, nil, nil, &ctx)
	if errors.Is(err, ErrAlreadyLoaded) {
		t.Fatalf("explicit Load broke later calls: %v", err)
	}
	if err == nil {
		_ = SCardReleaseContext(ctx)
	}
}

func TestDefaultLibraryPath_EnvVar(t *testing.T) {
	orig := os.Getenv("PCSCLITE_LIB_PATH")
	defer func() {
		if orig != "" {
			os.Setenv("PCSCLITE_LIB_PATH", orig)
		} else {
			os.Unsetenv("PCSCLITE_LIB_PATH")
		}
	}()

	os.Setenv("PCSCLITE_LIB_PATH", "/custom/libpcsclite.so")
	if got := defaultLibraryPath(); got != "/custom/libpcsclite.so" {
		t.Errorf("defaultLibraryPath() = %q, want %q", got, "/custom/libpcsclite.so")
	}

	os.Unsetenv("PCSCLITE_LIB_PATH")
	path := defaultLibraryPath()
	if path == "" {
		t.Error("defaultLibraryPath() should not be empty without env var")
	}
}

func TestPCSCLibraryPath_Purego(t *testing.T) {
	// For purego backend, should return the default path
	path := PCSCLibraryPath()
	if path == "" {
		t.Error("PCSCLibraryPath() should not be empty for purego backend")
	}
	t.Logf("PCSCLibraryPath() = %q", path)
}

func TestBackendInterface_Purego(t *testing.T) {
	// Verify the backend implements the interface
	b := getBackend()
	if b == nil {
		t.Fatal("backend is nil")
	}

	// Test that LibraryPath returns something
	path := b.LibraryPath()
	t.Logf("Backend.LibraryPath() = %q", path)

	// Test StringifyError
	errStr := b.StringifyError(0)
	t.Logf("StringifyError(0) = %q", errStr)
}
