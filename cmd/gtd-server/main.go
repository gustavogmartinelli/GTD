// Command gtd-server is the composition root: the only place that knows
// about every concrete layer and wires them together. It is
// "frameworks & drivers" in Clean Architecture terms.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gustavogmartinelli/gtd/internal/adapter/repository/sqlite"
	"github.com/gustavogmartinelli/gtd/internal/adapter/rest"
	"github.com/gustavogmartinelli/gtd/internal/usecase"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dbPath := flag.String("db", "gtd.db", "path to the SQLite database file")
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on -addr and exit 0 if healthy (for container HEALTHCHECK)")
	flag.Parse()

	if *healthcheck {
		if err := probe(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

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
	srv := &http.Server{
		Addr:              *addr,
		Handler:           rest.NewRouter(handler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Docker sends SIGTERM on stop/redeploy; finish in-flight requests
	// and close the database instead of dying mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gtd-server listening on %s (db: %s)", *addr, *dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// probe calls /healthz on the local server listening at addr.
func probe(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse -addr: %w", err)
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %s", resp.Status)
	}
	return nil
}
