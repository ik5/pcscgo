//go:build !(linux || freebsd || openbsd || netbsd)

package pcscgo

// Fails at compile time on any OS outside the supported list.
var _ = libpcscgo_unsupported_operating_system
