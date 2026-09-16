// Command gtd-server runs the GTD HTTP API.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gustavogmartinelli/gtd/internal/api"
	"github.com/gustavogmartinelli/gtd/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dbPath := flag.String("db", "gtd.db", "path to the SQLite database file")
	flag.Parse()

	s, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	handler := api.New(s).Router()

	log.Printf("gtd-server listening on %s (db: %s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
