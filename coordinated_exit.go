// This package is intended to simplify the management of global exit
// conditions, allowing any routine to signal all routines to exit, as well as
// optionally allowing os.Interrupt, and providing exit reasons as errors.
// This specifically does _not_ support resetting---once an exit flag has been
// set it won't go back.
package coordinated_exit

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
)

var shouldExit atomic.Bool
var exitReasons []error
var signalAlreadyHandled bool
var exitCond *sync.Cond
var mutex sync.Mutex

// Returns true if any routine or prior code as indicated that the program
// should exit.
func ShouldExit() bool {
	return (&shouldExit).Load()
}

// Returns the error causing the exit, or nil if no errors have been set.
func ExitReason() error {
	(&mutex).Lock()
	if len(exitReasons) == 0 {
		(&mutex).Unlock()
		return nil
	}
	if len(exitReasons) == 1 {
		(&mutex).Unlock()
		return exitReasons[0]
	}
	e := errors.Join(exitReasons...)
	(&mutex).Unlock()
	return e
}

// Sets the package-wide flag to exit, and allows any pending calls to
// WaitForExit() to return. Does not set an error, but won't clear any errors
// that have already been set by prior calls to ExitWithError().
func ExitWithoutError() {
	(&shouldExit).Store(true)
	exitCond.Broadcast()
}

// Sets the package-wide flag to exit, and adds the given error to the list of
// exit reasons to be returned by WaitForExit().
func ExitWithError(e error) {
	(&mutex).Lock()
	exitReasons = append(exitReasons, e)
	(&mutex).Unlock()
	ExitWithoutError()
}

// A simple wrapper around fmt.Errorf that calls ExitWithError.
func ExitWithErrorf(format string, args ...any) {
	e := fmt.Errorf(format, args...)
	ExitWithError(e)
}

// To be run in exactly one goroutine. Removes the signal handler and returns
// when either os.Interrupt occurs or when the exit flag is set for any other
// reason.
func waitForInterruptRoutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		// In a child routine, wait for the signal and set the exit flag when
		// it occurs or when c is closed. Either way, it's safe to call
		// ExitWithoutError()
		<-c
		ExitWithoutError()
	}()

	// The parent routine waits for exit simply to uninstall the signal handler
	// and close the channel when the exit occurs.
	WaitForExit()
	signal.Stop(c)
	close(c)
}

// Call this prior to WaitForExit() in order to set up handlers for
// os.Interrupt. If an os.Interrupt occurs, it will be equivalent to
// ExitWithoutError() being called, and cause WaitForExit() to return. The
// signal handler will be removed if ExitWithError() or ExitWithoutError() is
// called from any other context.
func ExitOnInterrupt() {
	(&mutex).Lock()
	if signalAlreadyHandled {
		// We already have a goroutine waiting for the interrupt.
		(&mutex).Unlock()
		return
	}

	go waitForInterruptRoutine()
	signalAlreadyHandled = true
	(&mutex).Unlock()
}

// Blocks until one of the following has occurred:
//   - ExitWithError(...) has been called
//   - ExitWithoutError() has been called
//   - An os.Interrupt signal was received and ExitOnInterrupt() has been
//     called.
//
// This is safe to call from multiple goroutines, or multiple times. It will
// immediately return if the exit has already been signalled. Returns any error
// passed to ExitWithError(...). If ExitWithError was called more than once,
// this will use errors.Join to combine them.
func WaitForExit() error {
	exitCond.L.Lock()
	for !(&shouldExit).Load() {
		exitCond.Wait()
	}
	exitCond.L.Unlock()
	return ExitReason()
}

func init() {
	exitCond = sync.NewCond(&mutex)
	exitReasons = make([]error, 0, 16)
}
