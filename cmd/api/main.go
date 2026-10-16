package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sianwa11/shazam-mwitu/api"
	"github.com/sianwa11/shazam-mwitu/store"
)

func main() {
	cfg := &api.Config{Address: ":" + getenv("PORT", "8080")}

	db, err := sql.Open("sqlite3", getenv("DB_DSN", "./songs.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := store.NewStore(db)

	server := api.NewServer(cfg, store)
	if err := server.Start(); err != nil {
		log.Fatalf("Critical server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
