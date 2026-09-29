package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sianwa11/shazam-mwitu/store"
)

type Config struct {
	Address string
}

type Server struct {
	config *Config
	store  *store.Store
	router *http.ServeMux
}

type UploadMetadata struct {
	Filename     string `json:"filename"`
	SizeBytes    int64  `json:"size_bytes"`
	DetectedType string `json:"detected_type"`
	DeclaredType string `json:"declared_type"`
	Extension    string `json:"extension"`
	TempPath     string `json:"temp_path"`
}

func NewServer(config *Config, store *store.Store) *Server {
	s := &Server{
		config: config,
		store:  store,
		router: http.NewServeMux(),
	}

	s.routes()

	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("GET /health", s.handleHealth())
	s.router.HandleFunc("POST /identify", s.handleIdentify())
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}
}

func (s *Server) handleIdentify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Must be a POST method
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit to 50MB
		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

		// Allow up to 25MB to stay in RAM
		if err := r.ParseMultipartForm(25 << 20); err != nil {
			log.Printf("Multipart parse error: %v", err)

			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		// Get the file
		file, header, err := r.FormFile("recording")
		if err != nil {
			http.Error(w, "Missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		buffer := make([]byte, 512)

		// Read few bytes into buffer
		_, err = file.Read(buffer)
		if err != nil && err != io.EOF {
			http.Error(w, "Unable to read file headers", http.StatusInternalServerError)
			return
		}

		// Reset pointer back to 0
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Unable to reset file stream", http.StatusInternalServerError)
			return
		}

		// verify file type
		contentType := http.DetectContentType(buffer)
		ext := filepath.Ext(header.Filename)
		if !strings.HasPrefix(contentType, "audio/") && contentType != "video/mp4" {
			log.Printf("Content-Type detection failed: detected=%q ext=%q filename=%q", contentType, ext, header.Filename)
			http.Error(w, "Must be an audio file", http.StatusBadRequest)
			return
		}

		// Create a temp file
		tempFile, err := os.CreateTemp("", "recordingsample-*"+ext)
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Copy into temp file
		_, err = io.Copy(tempFile, file)
		if err != nil {
			http.Error(w, "Couldn't copy file", http.StatusInternalServerError)
			return
		}

		meta := UploadMetadata{
			Filename:     header.Filename,
			SizeBytes:    header.Size,
			DetectedType: contentType,
			DeclaredType: header.Header.Get("Content-Type"),
			Extension:    ext,
			TempPath:     tempFile.Name(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(meta)

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
