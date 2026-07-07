package exercises

// Exercise 3 — FIX THE BUG: an early return that keeps the lock.
//
// Inventory.Reserve checks stock and decrements it — correctly guarded by a
// mutex, even. The author locked at the top and unlocked at the bottom, and
// it works... until the FIRST time a reservation fails. The error path
// returns without releasing the lock, and every call after that — any item,
// any goroutine — blocks forever on Lock(). One rare error path, whole
// service seized.
//
// This is Module 1 ex9's lesson (early return skips the cleanup) wearing a
// mutex: with a WaitGroup the damage was a hang at Wait(); with a mutex
// it's a poisoned lock that hangs EVERYONE, later, far from the bug.
//
// Your tasks:
//  1. Reproduce it mentally first: trace Reserve("bolts", 999) followed by
//     Reserve("bolts", 1). Where exactly does the second call stop?
//  2. Fix it with the idiom from the notes — the one that makes this whole
//     bug class impossible no matter how many return paths the function
//     grows later.
//
// Do not change signatures.

import (
	"fmt"
	"sync"
)

// Inventory tracks stock levels, safe for concurrent use. BUG: one path
// out of Reserve leaks the lock.
type Inventory struct {
	mu    sync.Mutex
	stock map[string]int // guarded by mu
}

// NewInventory returns an Inventory with the given starting stock.
func NewInventory(stock map[string]int) *Inventory {
	s := make(map[string]int, len(stock))
	for k, v := range stock {
		s[k] = v
	}
	return &Inventory{stock: s}
}

// Reserve atomically checks and decrements stock for an item.
func (inv *Inventory) Reserve(item string, qty int) error {
	inv.mu.Lock()
	if inv.stock[item] < qty {
		return fmt.Errorf("insufficient stock for %s: have %d, want %d",
			item, inv.stock[item], qty) // BUG: returns still holding the lock
	}
	inv.stock[item] -= qty
	inv.mu.Unlock()
	return nil
}

// Available returns the current stock for an item.
func (inv *Inventory) Available(item string) int {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	return inv.stock[item]
}
