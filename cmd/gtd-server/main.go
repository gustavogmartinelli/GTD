// Command gtd-server is the composition root: the only place that knows
// about every concrete layer and wires them together. It is
// "frameworks & drivers" in Clean Architecture terms.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gustavogmartinelli/gtd/internal/adapter/repository/sqlite"
	"github.com/gustavogmartinelli/gtd/internal/adapter/rest"
	"github.com/gustavogmartinelli/gtd/internal/usecase"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dbPath := flag.String("db", "gtd.db", "path to the SQLite database file")
	flag.Parse()

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewItemRepository(db)

	handler := rest.NewHandler(
		usecase.NewCaptureItemUseCase(repo),
		usecase.NewListItemsUseCase(repo),
		usecase.NewGetItemUseCase(repo),
		usecase.NewProcessItemUseCase(repo),
		usecase.NewUpdateItemUseCase(repo),
		usecase.NewCompleteItemUseCase(repo),
		usecase.NewDeleteItemUseCase(repo),
	)
	router := rest.NewRouter(handler)

	log.Printf("gtd-server listening on %s (db: %s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
