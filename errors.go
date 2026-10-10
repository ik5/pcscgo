// SPDX-License-Identifier: BSD-3-Clause

package pcscgo

import (
	"errors"
	"fmt"
)

// PCSCError represents a PC/SC error with its code and human-readable message.
type PCSCError struct {
	Code    ErrorCode
	Message string
}

// Error implements the error interface.
// Format: "[0x80100001] - An internal consistency check failed."
func (e *PCSCError) Error() string {
	return fmt.Sprintf("[%#x] - %s", uint32(e.Code), e.Message)
}

// Unwrap allows errors.Is/As to work with the underlying ErrorCode.
func (e *PCSCError) Unwrap() error {
	return e.Code
}

// IsSuccess returns true if the error code represents success (SCardSuccess).
func (e *PCSCError) IsSuccess() bool {
	return e.Code == SCardSuccess
}

// toError converts a raw Long return code from the library into an error.
// Returns nil for SCardSuccess, otherwise returns a *PCSCError with the
// code and the library's error message.
func toError(rc Long) error {
	code := CodeOf(rc)
	if code == SCardSuccess {
		return nil
	}
	return &PCSCError{
		Code:    code,
		Message: PCSCStringifyError(code),
	}
}

// LoadError represents an error loading the PC/SC library (purego mode).
type LoadError struct {
	Op  string // operation that failed (e.g., "dlopen", "dlsym")
	Err error  // underlying error
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("pcscgo: %s: %v", e.Op, e.Err)
}

func (e *LoadError) Unwrap() error {
	return e.Err
}

// IsNotLoaded returns true if the error indicates the library is not loaded.
func IsNotLoaded(err error) bool {
	var le *LoadError
	return errors.As(err, &le) && !errors.Is(err, ErrAlreadyLoaded)
}

// ErrAlreadyLoaded is returned when Load is called but a library is already loaded.
var ErrAlreadyLoaded = errors.New("pcscgo: library already loaded")

// ErrNotLoaded is returned when a function is called but the library is not loaded.
var ErrNotLoaded = errors.New("pcscgo: library not loaded")
