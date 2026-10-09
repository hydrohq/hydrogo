package server

import (
	"errors"
	"net"
	"sync"
)

type memoryListener struct {
	connections chan net.Conn
	closed      chan struct{}
	once        sync.Once
}

func newMemoryListener() *memoryListener {
	return &memoryListener{connections: make(chan net.Conn, 16), closed: make(chan struct{})}
}

func (listener *memoryListener) Accept() (net.Conn, error) {
	select {
	case connection := <-listener.connections:
		return connection, nil
	case <-listener.closed:
		return nil, net.ErrClosed
	}
}

func (listener *memoryListener) Close() error {
	listener.once.Do(func() { close(listener.closed) })
	return nil
}

func (listener *memoryListener) Addr() net.Addr { return memoryAddress("test") }

func (listener *memoryListener) dial() (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case listener.connections <- server:
		return client, nil
	case <-listener.closed:
		_ = client.Close()
		_ = server.Close()
		return nil, errors.New("in-memory listener is closed")
	}
}

type memoryAddress string

func (address memoryAddress) Network() string { return "in-memory" }
func (address memoryAddress) String() string  { return string(address) }
