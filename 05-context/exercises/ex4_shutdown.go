package exercises

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Exercise 4 — IMPLEMENT: graceful shutdown, the real four steps.
//
// ServeUntilCanceled runs an HTTP server until told to stop, then shuts
// it down gracefully. This is the exact shape of production `main()`s
// everywhere (including your shippio cmd/server):
//
//   1. serve on the given listener (srv.Serve(l)) in the background
//   2. block until ctx is canceled — that's the SIGTERM arriving
//      (in production ctx comes from signal.NotifyContext; the test
//      cancels it by hand, same mechanism)
//   3. drain: give in-flight requests up to drainTimeout to finish,
//      refuse new ones (one method on *http.Server does both — NOTES
//      Section 6)
//   4. return what the drain returned: nil if everything finished in
//      time, the error if the deadline won
//
// THE TRAP (the tests specifically check this): the drain needs its own
// context with drainTimeout. Think hard about which parent you derive
// it from. The ctx parameter is ALREADY CANCELED by the time you drain —
// a child of it is born dead, and your drain would abort instantly
// instead of waiting. NOTES Section 6, "fresh clock".
//
// Notes:
//   - srv.Serve(l) blocks until shutdown, returning http.ErrServerClosed
//     on a clean stop — run it in a goroutine, and don't report
//     ErrServerClosed as a failure
//   - do not call srv.ListenAndServe(); the tests need the listener they
//     pass in (it's how they get a free port)
//
// Do not change the signature.

// ORIGINAL (before fix): scaffold body was just panic("implement me").
// Solved correctly on the first attempt, including the deliberate trap —
// the drain context is derived from context.Background(), not from the
// already-canceled ctx parameter. A child of an already-dead ctx would
// be born dead, aborting the drain instantly instead of honoring
// drainTimeout.

// ServeUntilCanceled serves on l until ctx is canceled, then gracefully
// drains in-flight requests for up to drainTimeout. Returns nil on a
// clean drain, or the shutdown error if the deadline was exceeded.
func ServeUntilCanceled(ctx context.Context, srv *http.Server, l net.Listener, drainTimeout time.Duration) error {
	go func() { srv.Serve(l) }()

	<-ctx.Done()

	shCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	err := srv.Shutdown(shCtx)
	return err
}
