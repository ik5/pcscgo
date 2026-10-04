//go:build !(pcsc_static || pcsc_dynamic || pcsc_purego) || (pcsc_static && pcsc_dynamic) || (pcsc_static && pcsc_purego) || (pcsc_dynamic && pcsc_purego)

package pcscgo

// Fails at compile time unless exactly one of the tags pcsc_static,
// pcsc_dynamic or pcsc_purego is set.
var _ = libpcscgo_select_exactly_one_of_pcsc_static_pcsc_dynamic_pcsc_purego
