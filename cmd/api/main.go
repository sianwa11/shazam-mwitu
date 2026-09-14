package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sianwa11/shazam-mwitu/api"
)

func main() {
	cfg := &api.Config{Address: ":8080"}

	db, err := sql.Open("sqlite3", "./songs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := api.NewServer(cfg, db)
	if err := server.Start(); err != nil {
		log.Fatalf("Critical server error: %v", err)
	}
}
