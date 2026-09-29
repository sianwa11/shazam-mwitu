package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	identity "github.com/sianwa11/shazam-mwitu/Identity"
	"github.com/sianwa11/shazam-mwitu/fingerprint"
	"github.com/sianwa11/shazam-mwitu/pipeline"
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

type IdentifyResponse struct {
	Confidence string `json:"confidence"`
	Title      string `json:"title,omitempty"`
	Artist     string `json:"artist,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
	Votes      int    `json:"votes,omitempty"`
}

type ArtworkInfo struct {
	TrackName  string
	ArtistName string
	ArtworkURL string
}

type itunesResponse struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		TrackName     string `json:"trackName"`
		ArtistName    string `json:"artistName"`
		ArtworkURL100 string `json:"artworkUrl100"`
	} `json:"results"`
}

type NoMatch struct {
	Confidence string `json:"confidence"`
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

func writeJSON(w http.ResponseWriter, statusCode int, jsonVal any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(jsonVal)
}

func confidenceLabel(c identity.MatchConfidence) string {
	switch c {
	case identity.ConfidenceMatch:
		return "confident"
	case identity.PossibleMatch:
		return "possible"
	default:
		return "no_match"
	}
}

func lookupArtWork(title string) (ArtworkInfo, error) {
	params := url.Values{}
	params.Set("term", title)
	params.Set("media", "music")
	params.Set("entity", "song")
	params.Set("limit", "1")

	resp, err := http.Get("https://itunes.apple.com/search?" + params.Encode())
	if err != nil {
		return ArtworkInfo{}, fmt.Errorf("itunes request failed: %w", err)
	}
	defer resp.Body.Close()

	var result itunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ArtworkInfo{}, fmt.Errorf("decoding itunes response: %w", err)
	}

	if result.ResultCount == 0 {
		return ArtworkInfo{}, fmt.Errorf("no results found for %q", title)
	}

	track := result.Results[0]
	bigArtwork := strings.Replace(track.ArtworkURL100, "100x100", "600x600", 1)

	return ArtworkInfo{
		TrackName:  track.TrackName,
		ArtistName: track.ArtistName,
		ArtworkURL: bigArtwork,
	}, nil
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

		// Build the pipeline
		_, peaks, err := pipeline.BuildFingerprint(tempFile.Name())
		if err != nil {
			http.Error(w, "Failed to process audio", http.StatusInternalServerError)
			return
		}

		hashes := fingerprint.Hashing(peaks)

		scores := make(map[identity.Match]int)

		for _, h := range hashes {
			entries, err := s.store.LookupAddress(r.Context(), h.Address)
			if err != nil {
				continue
			}
			identity.AddMatches(scores, entries, h)
		}

		ranked := identity.RankMatches(scores)
		totalHashes := len(hashes)
		result := identity.BestMatch(ranked, totalHashes)

		if result.Confidence == identity.NoMatch {
			writeJSON(w, http.StatusOK, NoMatch{Confidence: "no_match"})
			return
		}

		// Both PossibleMatch and ConfidentMatch share this logic
		song, err := s.store.GetSong(r.Context(), result.Song.SongID)
		if err != nil {
			http.Error(w, "Failed to get song", http.StatusInternalServerError)
			return
		}

		artwork, err := lookupArtWork(song.Title)
		if err != nil {
			log.Printf("artwork lookup failed for %q: %v", song.Title, err)
			// fall back to your own stored title, no artwork
			artwork = ArtworkInfo{TrackName: song.Title}
		}

		response := IdentifyResponse{
			Confidence: confidenceLabel(result.Confidence),
			Title:      artwork.TrackName,
			Artist:     artwork.ArtistName,
			ArtworkURL: artwork.ArtworkURL,
			Votes:      result.Song.Count,
		}

		writeJSON(w, http.StatusOK, response)

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
