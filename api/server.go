package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

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
	s.router.HandleFunc("GET /songs", s.handleSongs())

	publicFS, err := fs.Sub(PublicFiles, "public")
	if err != nil {
		log.Fatal(err)
	}

	s.router.Handle("GET /", http.FileServer(http.FS(publicFS)))
}

func (s *Server) handleSongs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		songs, err := s.store.ListSongs(r.Context())
		if err != nil {
			log.Printf("listing songs: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list songs"})
			return
		}
		titles := make([]string, 0, len(songs))
		for _, song := range songs {
			titles = append(titles, song.Title)
		}
		writeJSON(w, http.StatusOK, map[string]any{"count": len(titles), "songs": titles})
	}
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

// normalizeName lowercases and keeps only letters and digits, so titles compare loosely.
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// lookupArtWork returns iTunes metadata only when a result matches the stored title.
// Stored titles without an artist ("08 Energy") are skipped: iTunes would guess an
// unrelated song with the same name.
func lookupArtWork(title string) (ArtworkInfo, error) {
	storedArtist, storedTrack, hasArtist := strings.Cut(title, " - ")
	if !hasArtist {
		return ArtworkInfo{}, fmt.Errorf("no artist in stored title %q, skipping lookup", title)
	}
	wantTrack := normalizeName(storedTrack)
	wantArtist := normalizeName(storedArtist)

	params := url.Values{}
	params.Set("term", title)
	params.Set("media", "music")
	params.Set("entity", "song")
	params.Set("limit", "5")

	resp, err := http.Get("https://itunes.apple.com/search?" + params.Encode())
	if err != nil {
		return ArtworkInfo{}, fmt.Errorf("itunes request failed: %w", err)
	}
	defer resp.Body.Close()

	var result itunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ArtworkInfo{}, fmt.Errorf("decoding itunes response: %w", err)
	}

	for _, track := range result.Results {
		gotTrack := normalizeName(track.TrackName)
		gotArtist := normalizeName(track.ArtistName)
		if gotTrack == "" || gotArtist == "" {
			continue
		}
		trackOK := strings.Contains(wantTrack, gotTrack)
		artistOK := strings.Contains(wantArtist, gotArtist) || strings.Contains(gotArtist, wantArtist)
		if trackOK && artistOK {
			bigArtwork := strings.Replace(track.ArtworkURL100, "100x100", "600x600", 1)
			return ArtworkInfo{
				TrackName:  track.TrackName,
				ArtistName: track.ArtistName,
				ArtworkURL: bigArtwork,
			}, nil
		}
	}

	return ArtworkInfo{}, fmt.Errorf("no matching itunes result for %q", title)
}

func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}
}

var allowedExtensions = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".mp4":  true,
	".m4a":  true,
	".webm": true,
	".ogg":  true,
	".flac": true,
}

func isAllowedExtension(ext string) bool {
	return allowedExtensions[strings.ToLower(ext)]
}

func (s *Server) handleIdentify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Must be a POST method
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit to 15MB
		r.Body = http.MaxBytesReader(w, r.Body, 15<<20)

		// Allow up to 10MB to stay in RAM
		if err := r.ParseMultipartForm(10 << 20); err != nil {
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
		ext := filepath.Ext(header.Filename)

		if !isAllowedExtension(ext) {
			log.Printf("rejected upload: filename=%q ext=%q", header.Filename, ext)
			http.Error(w, "Unsupported file type", http.StatusBadRequest)
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
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
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
