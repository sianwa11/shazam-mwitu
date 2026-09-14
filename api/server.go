package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Address string
}

type Server struct {
	config *Config
	db     *sql.DB
	router *http.ServeMux
}

func NewServer(config *Config, db *sql.DB) *Server {
	s := &Server{
		config: config,
		db:     db,
		router: http.NewServeMux(),
	}

	s.routes()

	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("GET /health", s.handleHealth())
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}
}

func (s *Server) Start() error {

	srv := &http.Server{
		Addr:         s.config.Address,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	shutDownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server gracefully...")

		// Provide a 5 second timer for active connections to finish processing
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutDownError <- srv.Shutdown(ctx)
	}()

	log.Printf("Starting server on %s\n", s.config.Address)

	// ListenAndServe always returns an error. http.ErrServerClosed is expected on normal shutdown.
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Await potential errors from the background shutdown goroutine
	if err := <-shutDownError; err != nil {
		return err
	}

	log.Println("Server stopped completely")
	return nil
}
