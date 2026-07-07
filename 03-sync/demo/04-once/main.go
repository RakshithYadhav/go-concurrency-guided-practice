// Demo 04: sync.Once — exactly once, and everyone waits.
//
//	go run ./03-sync/demo/04-once
//
// 20 goroutines all demand the config at the same moment. Watch:
//  1. "loading config..." prints exactly ONCE (guarantee 1: runs once)
//  2. every goroutine still gets a non-nil config — even the 19 "losers"
//     that arrived while the winner was mid-load. They BLOCKED inside
//     once.Do until loading finished (guarantee 2: everyone waits).
package main

import (
	"fmt"
	"sync"
	"time"
)

type Config struct{ dbURL string }

var (
	once   sync.Once
	config *Config
)

func loadConfig() {
	fmt.Println("loading config... (slow: 100ms)")
	time.Sleep(100 * time.Millisecond) // pretend to read a file
	config = &Config{dbURL: "postgres://localhost:5432/app"}
}

func GetConfig() *Config {
	once.Do(loadConfig)
	return config // never nil here — Do doesn't return until loadConfig finished
}

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := GetConfig()
			if cfg == nil {
				fmt.Println("goroutine", i, "got NIL — the everyone-waits guarantee broke!")
				return
			}
			fmt.Printf("goroutine %2d got config %s\n", i, cfg.dbURL)
		}()
	}
	wg.Wait()
	fmt.Println("done — count the 'loading config' lines above: exactly one")
}
