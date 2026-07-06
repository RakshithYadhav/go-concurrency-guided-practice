// Demo 01: main does not wait for goroutines.
//
// Run it a few times:
//
//	go run ./01-goroutines/demo/01-no-wait
//
// Most runs print nothing from the workers — main returns and the process
// exits before the goroutines get scheduled. Occasionally one sneaks a line
// out. That nondeterminism IS the lesson: without synchronization you have
// no guarantees, only luck.
package main

import "fmt"

func main() {
	for i := 0; i < 3; i++ {
		go func() {
			fmt.Println("worker", i, "reporting in")
		}()
	}
	fmt.Println("main is done") // then the process exits, workers or not
}
