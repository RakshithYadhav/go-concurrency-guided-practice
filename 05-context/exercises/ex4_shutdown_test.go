package exercises

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startServer wires up ServeUntilCanceled on a random free port and
// returns the base URL, the cancel that plays the role of SIGTERM, and
// a channel carrying ServeUntilCanceled's return value.
func startServer(t *testing.T, handler http.Handler, drainTimeout time.Duration) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv := &http.Server{Handler: handler}

	ret := make(chan error, 1)
	go func() { ret <- ServeUntilCanceled(ctx, srv, l, drainTimeout) }()
	time.Sleep(50 * time.Millisecond) // let the server come up

	return "http://" + l.Addr().String(), cancel, ret
}

func TestServe_HandlesRequestsWhileAlive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "pong")
	})
	base, cancel, ret := startServer(t, mux, time.Second)
	defer cancel()

	resp, err := http.Get(base + "/ping")
	if err != nil {
		t.Fatalf("server not serving: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "pong" {
		t.Fatalf("got %q, want %q", body, "pong")
	}

	cancel()
	select {
	case <-ret:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeUntilCanceled did not return after ctx cancel")
	}
}

func TestServe_DoesNotReturnBeforeCancel(t *testing.T) {
	base, cancel, ret := startServer(t, http.NewServeMux(), time.Second)
	defer cancel()
	_ = base

	select {
	case err := <-ret:
		t.Fatalf("returned before ctx was canceled: %v", err)
	case <-time.After(200 * time.Millisecond):
		// good: still serving
	}
	cancel()
	select {
	case <-ret:
	case <-time.After(2 * time.Second):
		t.Fatal("did not return after cancel")
	}
}

func TestServe_DrainsInFlightRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		io.WriteString(w, "finished")
	})
	base, cancel, ret := startServer(t, mux, 2*time.Second)

	// fire the in-flight request, then "SIGTERM" 100ms into it
	reqDone := make(chan string, 1)
	go func() {
		resp, err := http.Get(base + "/slow")
		if err != nil {
			reqDone <- "ERR: " + err.Error()
			return
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		reqDone <- string(body)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	// the in-flight request must complete — that's the drain
	select {
	case got := <-reqDone:
		if got != "finished" {
			t.Fatalf("in-flight request was killed instead of drained: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	// and ServeUntilCanceled must report a CLEAN drain. If it returns a
	// context error here, the drain context was derived from the already-
	// canceled signal ctx — the "born dead" trap.
	select {
	case err := <-ret:
		if err != nil {
			t.Fatalf("drain had 2s for a 300ms request but returned %v — check which parent the drain ctx derives from", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeUntilCanceled never returned")
	}

	// intake must be closed now
	if _, err := http.Get(base + "/slow"); err == nil {
		t.Fatal("server still accepting new requests after shutdown")
	} else if !strings.Contains(err.Error(), "refused") && !strings.Contains(err.Error(), "connect") {
		t.Logf("note: new request failed as expected with: %v", err)
	}
}

func TestServe_DeadlineWinsWhenDrainTooSlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/glacial", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
	})
	base, cancel, ret := startServer(t, mux, 100*time.Millisecond)

	go func() { http.Get(base + "/glacial") }() // in-flight, will NOT finish in time
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	cancel()
	select {
	case err := <-ret:
		if err == nil {
			t.Fatal("drain deadline was 100ms against a 3s request — want a non-nil error")
		}
		if took := time.Since(start); took > time.Second {
			t.Fatalf("gave up after %v — the 100ms drain deadline wasn't respected", took)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeUntilCanceled hung past its drain deadline")
	}
}
