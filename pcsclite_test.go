// SPDX-License-Identifier: BSD-3-Clause
package pcscgo

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestErrorCode_Error(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{"success", SCardSuccess, "PC/SC error 0x0"},
		{"internal", SCardFatalInternalError, "PC/SC error 0x80100001"},
		{"cancelled", SCardErrorCancelled, "PC/SC error 0x80100002"},
		{"custom", ErrorCode(0x80109999), "PC/SC error 0x80109999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorCode_AsLong(t *testing.T) {
	// On 386, Long is int32 and ErrorCode values >= 0x80000000 overflow int32.
	// Use a variable to avoid compile-time constant overflow check.
	fatalInternal := ErrorCode(0x80100001)
	tests := []struct {
		name string
		code ErrorCode
		want Long
	}{
		{"success", SCardSuccess, Long(0)},
		{"internal", SCardFatalInternalError, Long(fatalInternal)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.AsLong(); got != tt.want {
				t.Errorf("AsLong() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeOf(t *testing.T) {
	// Use helper to avoid compile-time constant overflow check on 32-bit platforms.
	rc1 := mustLong(0x80100001)
	rc2 := mustLong(0x80100002)
	tests := []struct {
		name string
		rc   Long
		want ErrorCode
	}{
		{"32-bit positive", rc1, SCardFatalInternalError},
		{"64-bit with high bits", rc2, SCardErrorCancelled},
		{"zero", Long(0), SCardSuccess},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeOf(tt.rc); got != tt.want {
				t.Errorf("CodeOf(%v) = %v, want %v", tt.rc, got, tt.want)
			}
		})
	}
}

func mustLong(v uint32) Long {
	return Long(v)
}

func TestPCSCError_Error(t *testing.T) {
	err := &PCSCError{
		Code:    SCardErrorNoSmartcard,
		Message: "No smart card inserted",
	}
	expected := "[0x8010000c] - No smart card inserted"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestPCSCError_Unwrap(t *testing.T) {
	err := &PCSCError{Code: SCardErrorTimeout}
	var target ErrorCode
	if !errors.As(err, &target) {
		t.Fatal("errors.As failed")
	}
	if target != SCardErrorTimeout {
		t.Errorf("unwrapped code = %v, want %v", target, SCardErrorTimeout)
	}
}

func TestPCSCError_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		err      *PCSCError
		expected bool
	}{
		{"success", &PCSCError{Code: SCardSuccess}, true},
		{"error", &PCSCError{Code: SCardErrorTimeout}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadError_Error(t *testing.T) {
	baseErr := errors.New("file not found")
	le := &LoadError{Op: "dlopen", Err: baseErr}
	expected := "pcscgo: dlopen: file not found"
	if got := le.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestLoadError_Unwrap(t *testing.T) {
	baseErr := errors.New("file not found")
	le := &LoadError{Op: "dlopen", Err: baseErr}
	if got := le.Unwrap(); got != baseErr {
		t.Errorf("Unwrap() = %v, want %v", got, baseErr)
	}
}

func TestIsNotLoaded(t *testing.T) {
	// Test with LoadError
	le := &LoadError{Op: "dlopen", Err: errors.New("not found")}
	if !IsNotLoaded(le) {
		t.Error("IsNotLoaded(LoadError) should be true")
	}

	// Test with properly wrapped LoadError
	wrapped := fmt.Errorf("wrapped: %w", le)
	if !IsNotLoaded(wrapped) {
		t.Error("IsNotLoaded(wrapped LoadError) should be true")
	}

	// Test with non-LoadError
	if IsNotLoaded(errors.New("some other error")) {
		t.Error("IsNotLoaded(other error) should be false")
	}

	// Already loaded means the library IS loaded
	if IsNotLoaded(&LoadError{Op: "load", Err: ErrAlreadyLoaded}) {
		t.Error("IsNotLoaded(ErrAlreadyLoaded) should be false")
	}
}

func TestDefaultLibraryPath(t *testing.T) {
	// Save original env
	orig := os.Getenv("PCSCLITE_LIB_PATH")
	defer func() {
		if orig != "" {
			os.Setenv("PCSCLITE_LIB_PATH", orig)
		} else {
			os.Unsetenv("PCSCLITE_LIB_PATH")
		}
	}()

	// Test with env var set
	os.Setenv("PCSCLITE_LIB_PATH", "/custom/path/libpcsclite.so")
	if got := defaultLibraryPath(); got != "/custom/path/libpcsclite.so" {
		t.Errorf("defaultLibraryPath() with env = %q, want %q", got, "/custom/path/libpcsclite.so")
	}

	// Test without env var (uses defaults based on GOOS for purego, empty for CGO)
	os.Unsetenv("PCSCLITE_LIB_PATH")
	path := defaultLibraryPath()
	// For purego backend, should return a default path; for CGO backends, returns empty
	t.Logf("defaultLibraryPath() = %q", path)
}

func TestPCSCLibraryPath(t *testing.T) {
	path := PCSCLibraryPath()
	t.Logf("PCSCLibraryPath() = %q", path)
	// For purego backend, should return a path
	// For CGO backends, returns empty string
}
