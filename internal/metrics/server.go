package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents a metrics HTTP server
type Server struct {
	server *http.Server
	port   int
}

// NewServer creates a new metrics server
func NewServer(port int) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)

	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		port: port,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	fmt.Printf("Starting metrics server on port %d\n", s.port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down metrics server...")
	return s.server.Shutdown(ctx)
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
