// SPDX-License-Identifier: BSD-3-Clause
//go:build (pcsc_static || pcsc_dynamic) && !cgo

package pcscgo

var _ = libpcscgo_static_and_dynamic_modes_require_CGO_ENABLED_1
