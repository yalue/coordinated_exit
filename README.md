A Library to Simplify Global Exit Conditions in Go
==================================================

Many of my programs follow a similar pattern: run some service until either a
signal occurs or a subroutine encounters an error. In either case, I would like
the remaining subroutines to have a chance to clean up before exiting
gracefully.

This package moves the boilerplate involved in maintaining the signal handler,
should-exit flag, and exit reason into its own package.


Usage
-----

The `coordinated_exit` package provides one primary function: `WaitForExit()`,
which blocks the caller until an exit condition occurs. The exit condition may
be set manually by any goroutine by calling `coordinated_exit.ExitWithError(err)`
or `coordinated_exit.ExitWithoutError()`. The `WaitForExit()` function will return
an error wrapping the error passed to `ExitWithError(..)`, or an error wrapping
it.

If you also want to exit when receiving an `os.Interrupt` signal, then call
`coordinated_exit.ExitOnInterrupt()` prior to calling `WaitForExit()`.

Example
-------

```go
import (
    "fmt"
    "github.com/yalue/coordinated_exit"
)

// Each worker runs until an error occurs, or until any single worker first
// reaches 9999 iterations (arbitrary example for illustration).
func runWorker() {
    maxIters := 9999
    for i := 0; i < maxIters; i++ {
        // End the loop if the exit flag is set
        if coordinated_exit.ShouldExit() {
            break
        }
        err := runOtherStuff()
        if err != nil {
            // Set the exit flag and the error that caused the exit
            coordinated_exit.ExitWithError(err)
            return
        }
    }
    // Max iterations reached, notify all other workers to exit, too. This
    // won't overwrite the exit flag or error if it was already set.
    coordinated_exit.ExitWithoutError()
}

func main() {
    coordinated_exit.ExitOnInterrupt()

    for i := 0; i < 10; i++ {
        go runWorker()
    }

    err := coordinated_exit.WaitForExit()
    if err != nil {
        fmt.Printf("Exited due to error: %s\n", err)
    } else {
        fmt.Printf("Exited without error.\n")
    }
}
```

