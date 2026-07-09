// Demo 03: a worker pool — 3 cooks, 9 tickets.
//
//	go run ./04-patterns/demo/03-workerpool
//
// Three long-lived workers all range over ONE jobs channel. Whoever is
// free grabs the next job — no job is assigned to a specific worker.
// The printout shows who grabbed what. Closing jobs is what sends the
// workers home; the WaitGroup + closer goroutine closes results (the
// multi-sender close pattern from Module 2).
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type result struct {
	worker, job, output int
}

func main() {
	jobs := make(chan int)
	results := make(chan result)
	var wg sync.WaitGroup

	// three cooks at the rail
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobs { // ends when jobs is closed
				time.Sleep(time.Duration(20+rand.Intn(40)) * time.Millisecond)
				results <- result{worker: id, job: j, output: j * j}
			}
		}(w)
	}

	// results has 3 senders → WaitGroup + closer goroutine closes it
	go func() {
		wg.Wait()
		close(results)
	}()

	// hand out 9 tickets, then close the rail
	go func() {
		for j := 1; j <= 9; j++ {
			jobs <- j
		}
		close(jobs)
	}()

	for r := range results {
		fmt.Printf("worker %d did job %d → %d\n", r.worker, r.job, r.output)
	}
	fmt.Println("jobs channel closed, all workers went home, results drained")
}
