// Path: internal/delivery/rest/server.go
package rest

import (
	"context"
	"net/http"
	"time"
)

// Server is the HTTP server for the read-only API.
type Server struct {
	httpServer *http.Server
}

// NewServer creates and configures a new API server.
func NewServer(port string, service dataService) *Server {
	modelHandlers := NewModelHandlers(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/models/", modelHandlers.GetModelByID) // Trailing slash handles sub-paths

	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + port,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}