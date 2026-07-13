package exercises

// Exercise 3 — GAUNTLET (25 min): build singleflight.
//
// The interview setup: 1,000 requests hit your server at the same
// moment, all asking for the same cache key. The database query behind
// it takes 100ms. If all 1,000 hit the database, it falls over. Make
// sure only ONE query actually runs — and all 1,000 callers get its
// result.
//
// That is singleflight (it ships in Google's groupcache). The rules:
//
//   - Do(key, fn): if no call for `key` is in flight, YOU are the
//     leader — run fn, and everyone who arrives with the same key
//     while you're running becomes a follower.
//   - Followers do NOT run fn. They wait for the leader's result and
//     return the same (value, err) the leader got. Errors are shared
//     exactly like values.
//   - Different keys never wait on each other.
//   - Once the leader finishes and the followers are served, the key
//     is forgotten: the NEXT Do with that key runs fn again. This is
//     dedup of CONCURRENT calls, not a cache.
//
// Think before you type:
//   - The shape: a map of in-flight calls, guarded (Module 3). Per
//     key: a place to park followers until the answer exists —
//     Module 2 gave you a primitive whose close wakes everyone.
//   - The ownership: who deletes the key from the map, and WHEN?
//     Get this wrong one way and a late arrival re-runs fn while the
//     leader still runs; wrong the other way and Do("k") after
//     completion returns a stale result instead of re-executing.
//   - The exits: what unblocks a follower? Exactly one thing. Make
//     sure it ALWAYS happens — even when fn returns an error.
//
// Do not change the signatures.

// Flight deduplicates concurrent calls per key: one execution,
// shared results.
type Flight struct {
	// your fields here
}

// NewFlight returns a ready-to-use Flight.
func NewFlight() *Flight {
	panic("implement me")
}

// Do executes fn for key, unless a call for key is already in flight —
// in that case it waits for the in-flight call and returns its result.
func (f *Flight) Do(key string, fn func() (string, error)) (string, error) {
	panic("implement me")
}
