// SPDX-License-Identifier: BSD-3-Clause
// scan - PC/SC Smart Card Scanner
//
// This program demonstrates how to use the pcscgo package to monitor smart card
// readers and card insertion/removal events. It is similar in functionality to
// the standard pcsc_scan utility that comes with pcsc-lite.
//
// The program uses SCardGetStatusChange which blocks until a reader state changes.
// To handle signals properly, we use SCardCancel from a signal handler to interrupt
// the blocking call, similar to how pcsc_scan works.
//
// BUILD MODES:
//   - purego:   go build -tags=pcsc_purego -o scan
//   - dynamic:  CGO_ENABLED=1 go build -tags=pcsc_dynamic -o dyn_scan
//   - static:   CGO_ENABLED=1 go build -tags=pcsc_static -o st_scan
//
// RUNNING:
//   ./scan                    # purego, uses default library path
//   ./scan -lib /path/lib.so  # purego, custom library path
//   ./dyn_scan                # dynamic linking
//   ./st_scan                 # static linking
//
// The program will:
// 1. Establish a context with the PC/SC resource manager
// 2. List all available readers
// 3. Monitor each reader for state changes (card insert/remove, reader add/remove)
// 4. Print human-readable events
// 5. Run until interrupted (Ctrl+C)

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ik5/pcscgo"
)

// readerStatus tracks the known state of each reader for better event detection.
type readerStatus struct {
	name           string
	wasConnected   bool // reader physically connected/available
	wasCardPresent bool // card present in reader
}

// scanner holds all state needed for the scanner loop.
type scanner struct {
	ctx       context.Context
	ctxHandle pcscgo.SCardContext
	timeout   time.Duration
	verbose   bool
	states    []pcscgo.SCardReaderState
	readers   []string
	statusMap map[uintptr]*readerStatus
	mu        sync.Mutex // protects states, readers, statusMap
	shutdown  bool
}

// main is the entry point. It parses flags, sets up signal handling,
// and runs the scanner loop.
func main() {
	// Parse command-line flags
	var (
		libPath = flag.String("lib", "", "Path to libpcsclite.so (purego mode only)")
		// Use INFINITE timeout (default) like pcsc_scan for proper event-driven monitoring.
		// Use -timeout=Nms for a custom finite timeout (not recommended).
		timeout = flag.Duration("timeout", -1, "Timeout for status change: -1=infinite (default), 0=1s poll, >0=custom ms")
		verbose = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Print mode information
	printModeInfo(*libPath)

	// Load the PC/SC library if needed (purego mode only).
	// For CGO modes, loadLibrary is a no-op.
	if err := loadLibrary(*libPath, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load library: %v\n", err)
		os.Exit(1)
	}

	// Create scanner and run
	s := &scanner{
		timeout:   *timeout,
		verbose:   *verbose,
		statusMap: make(map[uintptr]*readerStatus),
	}

	// Set up signal handling with SCardCancel for proper interruption
	// (like pcsc_scan does)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Run scanner in a goroutine so we can handle signals
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.run()
	}()

	// Signal handling loop
	for {
		select {
		case sig := <-sigCh:
			if sig == syscall.SIGQUIT {
				fmt.Println("\nReceived SIGQUIT, forcing exit...")
				s.cancel()
				// Give it a moment to clean up, then force exit
				select {
				case <-time.After(500 * time.Millisecond):
					fmt.Println("Force exit")
					os.Exit(1)
				case err := <-errCh:
					if err != nil {
						fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
						os.Exit(1)
					}
					os.Exit(0)
				}
			} else {
				fmt.Println("\nShutting down...")
				s.cancel()
			}
		case err := <-errCh:
			if err != nil {
				fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
}

// scanner runs the scanner logic.
func (s *scanner) run() error {
	// Step 1: Establish a context with the PC/SC resource manager.
	err := pcscgo.SCardEstablishContext(pcscgo.SCardScopeUser, nil, nil, &s.ctxHandle)
	if err != nil {
		return fmt.Errorf("SCardEstablishContext failed: %w", err)
	}
	if s.verbose {
		fmt.Printf("Established context: 0x%x\n", s.ctxHandle)
	}

	// Ensure we release the context when done.
	defer func() {
		if err := pcscgo.SCardReleaseContext(s.ctxHandle); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: SCardReleaseContext failed: %v\n", err)
		}
		if s.verbose {
			fmt.Println("Released context")
		}
	}()

	// Step 2: List all available readers.
	readers, err := s.listReaders()
	if err != nil {
		return fmt.Errorf("failed to list readers: %w", err)
	}
	s.readers = readers

	if len(readers) == 0 {
		fmt.Println("No readers found. Waiting for readers to be added...")
		s.readers = []string{}
	} else {
		fmt.Printf("Found %d reader(s):\n", len(readers))
		for i, r := range readers {
			fmt.Printf("  %d: %s\n", i, r)
		}
		fmt.Println()
	}

	// Step 3: Set up initial reader states for monitoring.
	s.setupStates()

	fmt.Println("Monitoring for card/reader events... (Press Ctrl+C to stop)")
	fmt.Println()

	// Monitoring loop
	for {
		err := s.waitForStateChange()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil // Graceful shutdown
			}
			return fmt.Errorf("waitForStateChange failed: %w", err)
		}

		s.mu.Lock()
		// Process all state changes
		for i := range s.states {
			s.processStateChange(i, &s.states[i])
		}

		// After processing, update CurrentState to EventState for next iteration
		for i := range s.states {
			s.states[i].CurrentState = s.states[i].EventState
		}

		// Check if reader list needs rebuilding (new/removed readers)
		s.rebuildIfNeeded()
		s.mu.Unlock()
	}
}

// cancel shuts down the scanner by canceling any blocking SCardGetStatusChange call.
func (s *scanner) cancel() {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return
	}
	s.shutdown = true
	// Use SCardCancel to interrupt any blocking SCardGetStatusChange call
	// This is how pcsc_scan handles signals properly.
	_ = pcscgo.SCardCancel(s.ctxHandle)
	s.mu.Unlock()
}

// waitForStateChange calls SCardGetStatusChange and handles errors.
func (s *scanner) waitForStateChange() error {
	s.mu.Lock()
	ctxHandle := s.ctxHandle
	states := s.states
	s.mu.Unlock()

	var timeoutDWord pcscgo.DWord
	if s.timeout > 0 {
		timeoutDWord = pcscgo.DWord(s.timeout.Milliseconds())
	} else if s.timeout == 0 {
		timeoutDWord = 1000 // 1 second polling
	} else {
		timeoutDWord = pcscgo.Infinite
	}

	// For CGO backends, we need to pin Go memory that contains pointers
	// passed to C. The states slice contains pointers (Reader, UserData).
	var pinner runtime.Pinner
	defer pinner.Unpin()

	s.mu.Lock()
	if len(states) > 0 {
		pinner.Pin(&states[0])
		for i := range states {
			if states[i].Reader != nil {
				pinner.Pin(states[i].Reader)
			}
			if states[i].UserData != nil {
				pinner.Pin(states[i].UserData)
			}
		}
	}
	s.mu.Unlock()

	err := pcscgo.SCardGetStatusChange(ctxHandle, timeoutDWord, states)
	if err != nil {
		var pcscErr *pcscgo.PCSCError
		if errors.As(err, &pcscErr) {
			// SCardErrorTimeout - normal for finite timeouts
			if pcscErr.Code == pcscgo.SCardErrorTimeout {
				return nil // Normal timeout, continue loop
			}
			// SCardErrorCancelled - normal when SCardCancel is called (Ctrl+C)
			if pcscErr.Code == pcscgo.SCardErrorCancelled {
				return context.Canceled // Graceful shutdown
			}
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return context.Canceled
		}
		return fmt.Errorf("SCardGetStatusChange failed: %w", err)
	}
	return nil
}

func (s *scanner) setupStates() {
	s.states = make([]pcscgo.SCardReaderState, len(s.readers))
	for i, reader := range s.readers {
		readerPtr, err := syscall.BytePtrFromString(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid reader name %q: %v\n", reader, err)
			continue
		}

		// states[i] corresponds to readers[i]; the PnP entry is always last.
		s.states[i] = pcscgo.SCardReaderState{
			Reader:       readerPtr,
			CurrentState: pcscgo.SCardStateUnaware, // We don't know the state yet
		}
	}

	// Also monitor for reader arrival/removal by adding a special "PnP" entry.
	pnpPtr, err := syscall.BytePtrFromString("\\\\?PnP?\\Notification")
	if err == nil {
		s.states = append(s.states, pcscgo.SCardReaderState{
			Reader:       pnpPtr,
			CurrentState: pcscgo.SCardStateUnaware,
		})
	}
}

// processStateChange handles a single reader's state change event.
func (s *scanner) processStateChange(i int, state *pcscgo.SCardReaderState) {
	// Check if state actually changed (the Changed flag alone is not a change)
	if state.EventState&^pcscgo.SCardStateChanged == state.CurrentState&^pcscgo.SCardStateChanged {
		return // No change
	}

	// Entries past the reader list are the PnP notification entry
	if i >= len(s.readers) {
		s.handlePnPNotification(state)
		return
	}
	readerName := s.readers[i]
	index := uintptr(i)

	// Get or create status tracker for this reader
	status, ok := s.statusMap[index]
	if !ok {
		status = &readerStatus{name: readerName}
		s.statusMap[index] = status
	}

	// Print the event based on what changed
	prev := state.CurrentState
	curr := state.EventState

	// --- READER CONNECTION STATE (physical reader hot-plug) ---

	// Reader became unavailable (physically disconnected or not accessible)
	if curr&pcscgo.SCardStateUnavailable != 0 && prev&pcscgo.SCardStateUnavailable == 0 {
		fmt.Printf("[%s] READER DISCONNECTED (unavailable)\n", readerName)
		status.wasConnected = false
		return
	}

	// Reader became available (physically connected)
	if prev&pcscgo.SCardStateUnavailable != 0 && curr&pcscgo.SCardStateUnavailable == 0 {
		fmt.Printf("[%s] READER CONNECTED (available)\n", readerName)
		status.wasConnected = true
		return
	}

	// --- CARD STATE (card insert/remove) ---
	// Only report card events if reader is connected
	if curr&pcscgo.SCardStateUnavailable == 0 {
		// Card inserted: was empty/unaware, now present
		if (prev&pcscgo.SCardStateEmpty != 0 || prev&pcscgo.SCardStateUnaware != 0) &&
			curr&pcscgo.SCardStatePresent != 0 && prev&pcscgo.SCardStatePresent == 0 {
			fmt.Printf("[%s] CARD INSERTED\n", readerName)
			s.printCardInfo(state)
			status.wasCardPresent = true
			return
		}

		// Card removed: was present, now empty
		if prev&pcscgo.SCardStatePresent != 0 && curr&pcscgo.SCardStateEmpty != 0 {
			fmt.Printf("[%s] CARD REMOVED\n", readerName)
			status.wasCardPresent = false
			return
		}

		// Card present but other state changed (e.g., ATR match, exclusive, etc.)
		if s.verbose && curr&pcscgo.SCardStatePresent != 0 {
			fmt.Printf("[%s] Card state changed: 0x%x -> 0x%x\n", readerName, prev, curr)
			s.printStateFlags(curr)
		}
	}

	// Other state changes (verbose only)
	if s.verbose {
		fmt.Printf("[%s] State changed: 0x%x -> 0x%x\n", readerName, prev, curr)
		s.printStateFlags(curr)
	}
}

// handlePnPNotification reports PnP (Plug and Play) notifications for reader hot-plug.
// pcscd only sets the Changed flag (plus an event counter in the upper 16 bits) on
// this entry; which readers were added or removed is found by rebuildIfNeeded.
func (s *scanner) handlePnPNotification(state *pcscgo.SCardReaderState) {
	if s.verbose {
		fmt.Printf("[PnP] State changed: 0x%x -> 0x%x\n", state.CurrentState, state.EventState)
		s.printStateFlags(state.EventState)
	}
}

// rebuildIfNeeded checks if the reader list has changed and rebuilds state array if needed.
func (s *scanner) rebuildIfNeeded() {
	// The PnP notification entry is always the last state
	if len(s.states) <= len(s.readers) || s.states[len(s.states)-1].EventState&pcscgo.SCardStateChanged == 0 {
		return
	}

	// PnP notification with change - re-enumerate readers
	readers, err := s.listReaders()
	if err != nil {
		if s.verbose {
			fmt.Printf("Warning: failed to re-enumerate readers: %v\n", err)
		}
		return
	}

	// Check if reader list actually changed
	if len(readers) != len(s.readers) || !equalStringSlices(readers, s.readers) {
		fmt.Printf("Reader list changed, re-enumerating (%d -> %d readers)\n", len(s.readers), len(readers))
		for _, r := range readers {
			if !slices.Contains(s.readers, r) {
				fmt.Printf("READER ADDED: %s\n", r)
			}
		}
		for _, r := range s.readers {
			if !slices.Contains(readers, r) {
				fmt.Printf("READER REMOVED: %s\n", r)
			}
		}
		s.readers = readers
		s.setupStates()
		// Re-initialize statusMap
		s.statusMap = make(map[uintptr]*readerStatus)
		for i, state := range s.states {
			name := "PnP Notification"
			if i < len(s.readers) {
				name = s.readers[i]
			}
			s.statusMap[uintptr(i)] = &readerStatus{
				name:           name,
				wasConnected:   state.EventState&pcscgo.SCardStateUnavailable == 0,
				wasCardPresent: state.EventState&pcscgo.SCardStatePresent != 0,
			}
		}
	}
}

// listReaders calls SCardListReaders to get all reader names.
func (s *scanner) listReaders() ([]string, error) {
	n, err := pcscgo.SCardListReaders(s.ctxHandle, "", nil)
	if err != nil {
		return nil, fmt.Errorf("SCardListReaders (query size) failed: %w", err)
	}
	if n == 0 {
		if s.verbose {
			fmt.Println("No readers found")
		}
		return []string{}, nil
	}

	buf := make([]byte, n)
	n, err = pcscgo.SCardListReaders(s.ctxHandle, "", buf)
	if err != nil {
		return nil, fmt.Errorf("SCardListReaders (read) failed: %w", err)
	}

	return parseMultiString(buf[:n]), nil
}

// printCardInfo prints detailed information about the inserted card.
func (s *scanner) printCardInfo(state *pcscgo.SCardReaderState) {
	if state.ATRLength > 0 {
		atr := state.ATR[:state.ATRLength]
		fmt.Printf("  ATR: %s\n", s.formatHex(atr))
		if s.verbose {
			fmt.Printf("  ATR Length: %d bytes\n", state.ATRLength)
		}
	}
	if state.EventState&pcscgo.SCardStatePresent != 0 {
		if s.verbose {
			fmt.Println("  Card is present and powered")
		}
	}
}

// printStateFlags prints human-readable state flags for debugging.
func (s *scanner) printStateFlags(state pcscgo.DWord) {
	flags := []string{}
	if state&pcscgo.SCardStateUnaware != 0 {
		flags = append(flags, "Unaware")
	}
	if state&pcscgo.SCardStateIgnore != 0 {
		flags = append(flags, "Ignore")
	}
	if state&pcscgo.SCardStateChanged != 0 {
		flags = append(flags, "Changed")
	}
	if state&pcscgo.SCardStateUnknown != 0 {
		flags = append(flags, "Unknown")
	}
	if state&pcscgo.SCardStateUnavailable != 0 {
		flags = append(flags, "Unavailable")
	}
	if state&pcscgo.SCardStateEmpty != 0 {
		flags = append(flags, "Empty")
	}
	if state&pcscgo.SCardStatePresent != 0 {
		flags = append(flags, "Present")
	}
	if state&pcscgo.SCardStateATRMatch != 0 {
		flags = append(flags, "ATRMatch")
	}
	if state&pcscgo.SCardStateExclusive != 0 {
		flags = append(flags, "Exclusive")
	}
	if state&pcscgo.SCardStateInUse != 0 {
		flags = append(flags, "InUse")
	}
	if state&pcscgo.SCardStateMute != 0 {
		flags = append(flags, "Mute")
	}
	if state&pcscgo.SCardStateUnpowered != 0 {
		flags = append(flags, "Unpowered")
	}
	fmt.Printf("  State flags: %s\n", strings.Join(flags, " | "))
}

// formatHex formats a byte slice as hex string with spaces.
func (s *scanner) formatHex(data []byte) string {
	if len(data) == 0 {
		return "(empty)"
	}
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, " ")
}

// printModeInfo prints information about which build mode is running.
func printModeInfo(libPath string) {
	fmt.Println("=== PC/SC Smart Card Scanner ===")

	path := pcscgo.PCSCLibraryPath()
	if path != "" {
		fmt.Printf("Mode: purego (runtime library loading)\n")
		if libPath != "" {
			fmt.Printf("Library: %s (from -lib flag)\n", libPath)
		} else {
			fmt.Printf("Library: %s (default)\n", path)
		}
	} else {
		fmt.Println("Mode: CGO (compile-time linking)")
	}
	fmt.Println()
}

// equalStringSlices checks if two string slices are equal.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// parseMultiString converts a PC/SC multi-string to a Go string slice.
// A multi-string is: "Reader1\0Reader2\0Reader3\0\0"
func parseMultiString(data []byte) []string {
	var result []string
	var current strings.Builder

	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				break
			}
		} else {
			current.WriteByte(data[i])
		}
	}
	return result
}
