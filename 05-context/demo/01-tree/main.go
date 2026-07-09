// Demo 01: contexts form a tree — cancellation flows DOWN, never up.
//
//	go run ./05-context/demo/01-tree
//
// Three levels: root → request → dbcall. We cancel the MIDDLE one (the
// request). Watch: the dbcall (child) stops immediately, the root
// (parent) keeps running, untouched. Down, not up.
package main

import (
	"context"
	"fmt"
	"time"
)

// watch reports when its context dies (or says it stayed alive).
func watch(name string, ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			fmt.Printf("  %-8s CANCELED (%v)\n", name, ctx.Err())
		case <-time.After(500 * time.Millisecond):
			fmt.Printf("  %-8s still alive after 500ms — cancellation never reached it\n", name)
		}
	}()
}

func main() {
	root := context.Background()
	rootCtx, rootCancel := context.WithCancel(root)
	defer rootCancel()

	reqCtx, reqCancel := context.WithCancel(rootCtx) // child of root
	defer reqCancel()

	dbCtx, dbCancel := context.WithCancel(reqCtx) // child of request
	defer dbCancel()

	watch("root", rootCtx)
	watch("request", reqCtx)
	watch("dbcall", dbCtx)

	fmt.Println("tree: root → request → dbcall")
	fmt.Println("canceling the MIDDLE (request) in 100ms...")
	time.Sleep(100 * time.Millisecond)
	reqCancel()

	time.Sleep(600 * time.Millisecond) // let the watchers report
	fmt.Println("\nlesson: canceling request killed dbcall (its child),")
	fmt.Println("but root (its parent) never noticed. Down, not up.")
}
