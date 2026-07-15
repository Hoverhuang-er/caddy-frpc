# Task 2: Listener Adapter (listener.go)

**Files:**
- Create: `listener.go`
- Test: `frpc_test.go` (add tests)

## Implementation

Create `listener.go` with:

```go
package frpc

import (
	"net"
	"sync"
	"sync/atomic"
)

// frpcListener adapts frpc work connections into a net.Listener.
type frpcListener struct {
	name   string
	ch     chan net.Conn
	addr   net.Addr
	closed atomic.Bool
}

func newFRPCListener(name, addr string) *frpcListener {
	return &frpcListener{
		name: name,
		ch:   make(chan net.Conn, 8),
		addr: resolveAddr(addr),
	}
}

func (l *frpcListener) Accept() (net.Conn, error) {
	conn, ok := <-l.ch
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (l *frpcListener) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return net.ErrClosed
	}
	close(l.ch)
	return nil
}

func (l *frpcListener) Addr() net.Addr { return l.addr }
func (l *frpcListener) Name() string   { return l.name }
func (l *frpcListener) ConnChan() chan<- net.Conn { return l.ch }

func resolveAddr(addr string) net.Addr {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
	}
	return tcpAddr
}

// multiListener aggregates multiple frpcListeners into one net.Listener.
type multiListener struct {
	sub    map[string]*frpcListener
	ch     chan net.Conn
	done   chan struct{}
	once   sync.Once
}

func newMultiListener(listeners map[string]*frpcListener) *multiListener {
	ml := &multiListener{
		sub:  listeners,
		ch:   make(chan net.Conn, 16),
		done: make(chan struct{}),
	}
	for _, ln := range listeners {
		ln := ln
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				select {
				case ml.ch <- conn:
				case <-ml.done:
					return
				}
			}
		}()
	}
	return ml
}

func (ml *multiListener) Accept() (net.Conn, error) {
	select {
	case conn := <-ml.ch:
		return conn, nil
	case <-ml.done:
		return nil, net.ErrClosed
	}
}

func (ml *multiListener) Close() error {
	ml.once.Do(func() {
		close(ml.done)
		for _, ln := range ml.sub {
			ln.Close()
		}
	})
	return nil
}

func (ml *multiListener) Addr() net.Addr {
	for _, ln := range ml.sub {
		return ln.Addr()
	}
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}
```

## Tests (in frpc_test.go)

```go
package frpc

import (
	"net"
	"testing"
)

func TestFRPCListenerAcceptClose(t *testing.T) {
	l := newFRPCListener("test-proxy", ":8080")
	defer l.Close()

	done := make(chan struct{})
	go func() {
		conn1, conn2 := net.Pipe()
		defer conn2.Close()
		l.ConnChan() <- conn1
		done <- struct{}{}
	}()

	accepted, err := l.Accept()
	if err != nil {
		t.Fatalf("Accept() error: %v", err)
	}
	accepted.Close()
	<-done

	if l.Addr().String() != "127.0.0.1:8080" {
		t.Fatalf("expected 127.0.0.1:8080, got %s", l.Addr().String())
	}
}

func TestFRPCListenerCloseUnblocksAccept(t *testing.T) {
	l := newFRPCListener("test-proxy", ":8080")
	go l.Close()
	_, err := l.Accept()
	if err != net.ErrClosed {
		t.Fatalf("expected net.ErrClosed, got %v", err)
	}
}

func TestFRPCListenerName(t *testing.T) {
	l := newFRPCListener("web", ":8080")
	if l.Name() != "web" {
		t.Fatalf("expected web, got %s", l.Name())
	}
}

func TestMultiListener(t *testing.T) {
	ln1 := newFRPCListener("proxy1", ":8080")
	ln2 := newFRPCListener("proxy2", ":9090")
	ml := newMultiListener(map[string]*frpcListener{"p1": ln1, "p2": ln2})
	defer ml.Close()

	done := make(chan struct{})
	go func() {
		conn1, conn2 := net.Pipe()
		defer conn2.Close()
		ln1.ConnChan() <- conn1
		done <- struct{}{}
	}()

	conn, err := ml.Accept()
	if err != nil {
		t.Fatalf("multiListener Accept() error: %v", err)
	}
	conn.Close()
	<-done
}
```

## Acceptance

- All 4 tests PASS (3 frpcListener + 1 multiListener)
- `go vet ./...` passes
- Committed with message `"feat: add frpcListener adapter (workConn -> net.Listener)"`
