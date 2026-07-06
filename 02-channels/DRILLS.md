# Module 2 — Predict-the-Output Drills

For each snippet, write down — **before running anything** — exactly what
happens: the output, a deadlock (`fatal error: all goroutines are asleep`),
a panic (which message?), or a hang. Then justify it from the axioms table.
We go through your answers at review time. (These are the classic shapes
interviewers put on a whiteboard; prediction-without-running is the skill
being tested.)

Each one is tiny on purpose. Assume each is the complete `main` package.

---

### Drill 1

```go
func main() {
	ch := make(chan int)
	ch <- 1
	fmt.Println(<-ch)
}
```

### Drill 2

```go
func main() {
	ch := make(chan int, 1)
	ch <- 1
	fmt.Println(<-ch)
}
```

### Drill 3

```go
func main() {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)
	for v := range ch {
		fmt.Println(v)
	}
}
```

### Drill 4

```go
func main() {
	ch := make(chan int)
	close(ch)
	v, ok := <-ch
	fmt.Println(v, ok)
	v, ok = <-ch
	fmt.Println(v, ok)
}
```

### Drill 5

```go
func main() {
	ch := make(chan int)
	close(ch)
	ch <- 1
}
```

### Drill 6

```go
func main() {
	var ch chan int   // note: var, not make
	select {
	case v := <-ch:
		fmt.Println(v)
	default:
		fmt.Println("default")
	}
}
```

### Drill 7

```go
func main() {
	var ch chan int   // nil again — but no default this time
	<-ch
}
```

### Drill 8

```go
func main() {
	ch := make(chan int)
	go func() {
		ch <- 42
	}()
	fmt.Println(<-ch)
	fmt.Println(<-ch)
}
```

### Drill 9

```go
func main() {
	ch := make(chan string)
	go func() {
		ch <- "first"
		ch <- "second"
		close(ch)
	}()
	fmt.Println(<-ch)
	for v := range ch {
		fmt.Println(v)
	}
}
```

### Drill 10

```go
func main() {
	done := make(chan struct{})
	go func() {
		fmt.Println("working")
		close(done)
	}()
	<-done
	fmt.Println("main exits")
}
```

*(Drill 10 also introduces a real idiom worth noticing: a `chan struct{}`
used purely as a completion signal, where close IS the message — no value
ever sent. You'll meet it constantly in real code and in Module 5.)*
