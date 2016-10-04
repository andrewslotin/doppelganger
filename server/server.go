package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

// ErrNotStarted is an error returned by (*Server).Shutdown() if the server was not started yet.
var ErrNotStarted = errors.New("server not running")

// Server is a type representing an HTTP server.
type Server struct {
	Addr string

	ln net.Listener
	mu sync.Mutex
}

// New returns an unstarted *Server instance that serves connections on provided host:port.
func New(host string, port int) *Server {
	return &Server{Addr: fmt.Sprintf("%s:%d", host, port)}
}

// Run starts the server and spawns a goroutine that accepts incoming connections and handles them using http.Handler.
func (srv *Server) Run(h http.Handler) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %s", srv.Addr, err)
	}
	srv.ln = ln

	go func(srv *http.Server, ln net.Listener) {
		if err := srv.Serve(ln); err != nil {
			log.Fatal(err)
		}
	}(&http.Server{Handler: h}, srv.ln)

	return nil
}

// Shutdown terminates running server.
func (srv *Server) Shutdown() error {
	if srv.ln == nil {
		return ErrNotStarted
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if err := srv.ln.Close(); err != nil {
		return err
	}

	srv.ln = nil

	return nil
}
