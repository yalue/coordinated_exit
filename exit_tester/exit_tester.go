// This is simply a quick program that responds to Ctrl+C or exits on its own
// after a while.
//
// This tests a few things:
//  1. That one worker can signal all others to exit properly.
//  2. That signal handlers cause everything to exit when the signal is
//     received.
//  3. After the os.Interrupt signal is received, subsequent interrupts go back
//     to working in the default manner (i.e., killing the program).
package main

import (
	"fmt"
	"github.com/yalue/coordinated_exit"
	"math/rand"
	"time"
)

// Each worker invokes time.Sleep(1 * time.Second) n times, unless it's already
// been signalled to exit. The first worker to run n iterations will signal the
// remaining workers to exit.
func runWorker(workerID, n int) {
	fmt.Printf("Worker ID %d should sleep for %d seconds.\n", workerID, n)
	i := 0
	for ; i < n; i++ {
		if coordinated_exit.ShouldExit() {
			break
		}
		time.Sleep(1 * time.Second)
	}
	coordinated_exit.ExitWithoutError()
	fmt.Printf("Worker ID %d exited after %d/%d iterations.\n", workerID, i+1,
		n)
}

func main() {
	coordinated_exit.ExitOnInterrupt()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	numRoutines := 10 + rng.Intn(10)
	fmt.Printf("Starting %d routines.\n", numRoutines)

	for i := 0; i < numRoutines; i++ {
		go runWorker(i, 10+rng.Intn(10))
	}

	fmt.Printf("Main routine waiting for exit...\n")
	e := coordinated_exit.WaitForExit()
	if e != nil {
		fmt.Printf("WaitForExit() returned error %s\n", e)
	} else {
		fmt.Printf("WaitForExit() returned no error.\n")
	}
	fmt.Printf("Sleeping 10 more seconds...\n")
	time.Sleep(10 * time.Second)
	fmt.Printf("All done!\n")
}
