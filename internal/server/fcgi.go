package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// ServeFastCGI listens for FastCGI responder connections and drains active
// requests when its context is canceled.
func ServeFastCGI(ctx context.Context, network, address string, config Config) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	return ServeFastCGIListener(ctx, listener, NewHandler(config), config.ShutdownTimeout)
}

// ServeFastCGIListener serves FastCGI requests on an existing net.Listener and drains active
// requests when its context is canceled.
func ServeFastCGIListener(ctx context.Context, listener net.Listener, handler http.Handler, timeout time.Duration) error {
	tracked := &trackedListener{Listener: listener, conns: make(map[net.Conn]struct{})}
	gate := &requestGate{}
	serveErr := make(chan error, 1)
	go func() { serveErr <- ServeFCGI(tracked, gate.wrap(handler)) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	gate.stop()
	_ = tracked.Close()
	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	case <-time.After(shutdownTimeout):
		tracked.closeConnections()
		return errors.New("FastCGI listener did not stop")
	}

	if timeout <= 0 {
		timeout = shutdownTimeout
	}
	drained := make(chan struct{})
	go func() { gate.wait(); close(drained) }()
	select {
	case <-drained:
	case <-time.After(timeout):
	}
	tracked.closeConnections()
	<-drained
	return nil
}

// ServeFastCGIStdin serves FastCGI requests from standard input (such as when spawned
// by Apache mod_fcgid or mod_fastcgi). If os.Stdin can be converted to a net.Listener,
// it uses ServeFastCGIListener for graceful shutdown; otherwise it falls back to ServeFCGI(nil, handler).
func ServeFastCGIStdin(ctx context.Context, handler http.Handler, timeout time.Duration) error {
	listener, err := net.FileListener(os.Stdin)
	if err != nil {
		return ServeFCGI(nil, handler)
	}
	return ServeFastCGIListener(ctx, listener, handler, timeout)
}

type requestGate struct {
	mu      sync.Mutex
	stopped bool
	active  sync.WaitGroup
}

func (g *requestGate) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		if g.stopped {
			g.mu.Unlock()
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		g.active.Add(1)
		g.mu.Unlock()
		defer g.active.Done()
		next.ServeHTTP(w, r)
	})
}

func (g *requestGate) stop() {
	g.mu.Lock()
	g.stopped = true
	g.mu.Unlock()
}

func (g *requestGate) wait() { g.active.Wait() }

type trackedListener struct {
	net.Listener
	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func (l *trackedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tracked := &trackedConn{Conn: conn, listener: l}
	l.mu.Lock()
	l.conns[tracked] = struct{}{}
	l.mu.Unlock()
	return tracked, nil
}

func (l *trackedListener) closeConnections() {
	l.mu.Lock()
	conns := make([]net.Conn, 0, len(l.conns))
	for conn := range l.conns {
		conns = append(conns, conn)
	}
	l.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
}

type trackedConn struct {
	net.Conn
	listener *trackedListener
	once     sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		c.listener.mu.Lock()
		delete(c.listener.conns, c)
		c.listener.mu.Unlock()
	})
	return err
}
