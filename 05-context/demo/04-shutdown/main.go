// Demo 04: graceful shutdown of a real HTTP server, mid-request.
//
//	go run ./05-context/demo/04-shutdown
//
// The server has a /slow endpoint that takes 2 seconds. We:
//  1. start the server,
//  2. fire a /slow request,
//  3. trigger shutdown 500ms in (standing in for SIGTERM — same context
//     mechanism signal.NotifyContext would cancel),
//  4. watch: the in-flight request FINISHES (drain), a new request is
//     REFUSED (intake closed), then the process exits cleanly.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  server: /slow started (takes 2s)")
		time.Sleep(2 * time.Second)
		fmt.Fprintln(w, "slow work done")
		fmt.Println("  server: /slow finished and responded")
	})

	srv := &http.Server{Addr: "127.0.0.1:8093", Handler: mux}

	// In production this ctx comes from signal.NotifyContext(SIGTERM/SIGINT).
	// Here a 500ms timer plays the role of the operator pressing Ctrl+C.
	ctx, trigger := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("== pretend SIGTERM arrives now ==")
		trigger()
	}()

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("serve error:", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // let the listener come up

	// fire the in-flight request
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := http.Get("http://127.0.0.1:8093/slow")
		if err != nil {
			fmt.Println("  client: in-flight request FAILED:", err)
			return
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("  client: in-flight request got: %q\n", string(body[:len(body)-1]))
	}()

	<-ctx.Done() // step 1: heard the "signal"

	fmt.Println("== draining: stop intake, finish in-flight (10s budget) ==")
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // fresh clock!
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil { // steps 2+3
		fmt.Println("drain deadline lost, forced exit:", err)
	} else {
		fmt.Println("== drained cleanly ==")
	}

	// prove intake is closed
	if _, err := http.Get("http://127.0.0.1:8093/slow"); err != nil {
		fmt.Println("  client: NEW request after shutdown: refused ✔")
	}

	<-done // make sure the in-flight client printed its result
	fmt.Println("exit. the in-flight request finished; new work was refused.")
}
