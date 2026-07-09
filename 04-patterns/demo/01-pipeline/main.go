// Demo 01: a 3-stage pipeline — generate → square → print.
//
//	go run ./04-patterns/demo/01-pipeline
//
// Each stage is a goroutine. Each arrow between them is a channel. Watch
// the narration: stages overlap (while square works on one value, generate
// is already handing over the next), and shutdown cascades — generate
// closes, so square's range ends, so square closes, so main's range ends.
package main

import "fmt"

func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			fmt.Printf("  generate: sending %d\n", n)
			out <- n
		}
		fmt.Println("  generate: done, closing my output")
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			fmt.Printf("  square:   got %d, sending %d\n", v, v*v)
			out <- v * v
		}
		fmt.Println("  square:   input closed, closing my output")
	}()
	return out
}

func main() {
	fmt.Println("pipeline: generate → square → main")
	for result := range square(generate(1, 2, 3, 4, 5)) {
		fmt.Printf("main:     received %d\n", result)
	}
	fmt.Println("main:     pipeline drained, everything shut down clean")
}
