package exercises

import "sync"

// Exercise 2 — FIX THE BUG: check-then-act on a shared cache (TOCTOU).
//
// PriceCache memoizes slow price lookups: first request for a symbol calls
// fetch, later requests hit the cache. Under one goroutine it's perfect.
// Under many, TWO distinct things go wrong — run the test and identify both:
//
//  1. The map itself is touched concurrently, unsynchronized. You know what
//     Go does about that (ex2 of Module 1: race report, and often the
//     runtime's own "concurrent map read and map write" crash).
//  2. Subtler: even if the map were magically safe, the LOGIC races.
//     "Check (miss) ... then act (fetch + store)" is two steps — N
//     goroutines can ALL see a miss before any of them stores, so the
//     expensive fetch runs N times for one symbol. Time-Of-Check-To-
//     Time-Of-Use (TOCTOU) — the same shape as sync.Once's naive
//     `if !done` bug in the notes.
//
// Your tasks:
//  1. Fix it so it's race-free AND fetch runs AT MOST ONCE per symbol —
//     the test counts calls. Think about what the lock must cover for the
//     count to be exactly 1: is guarding the map reads/writes enough, or
//     does the whole check-fetch-store need to be one critical section?
//  2. Be ready to defend the cost: with your fix, what happens to requests
//     for OTHER symbols while one symbol's slow fetch holds the lock?
//     (That tradeoff has a proper fix — per-key locking — in Module 6.
//     Here, correctness first.)
//
// Do not change signatures.

// PriceCache memoizes fetch results per symbol. BUG: unsynchronized
// check-then-act on a shared map.
type PriceCache struct {
	mu     sync.Mutex
	prices map[string]float64
}

// NewPriceCache returns an empty cache.
func NewPriceCache() *PriceCache {
	return &PriceCache{prices: make(map[string]float64)}
}

// GetOrFetch returns the cached price for symbol, calling fetch on a miss.
func (c *PriceCache) GetOrFetch(symbol string, fetch func(symbol string) float64) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if price, ok := c.prices[symbol]; ok {
		return price
	}
	price := fetch(symbol)
	c.prices[symbol] = price
	return price
}
