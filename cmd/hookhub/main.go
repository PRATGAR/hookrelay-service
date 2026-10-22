package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"hookhub/internal/api"
	"hookhub/internal/dispatch"
	"hookhub/internal/guard"
	"hookhub/internal/queue"
	"hookhub/internal/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	migrate := flag.Bool("migrate", false, "apply the database schema and exit")
	flag.Parse()

	pepper := env("KEY_PEPPER", "")
	if pepper == "" {
		log.Fatal("KEY_PEPPER is not set")
	}
	guard.UseKeyPepper(pepper)
	rand.Seed(time.Now().Unix())

	db, err := sql.Open("postgres", env("DATABASE_URL", "postgres://hookhub@localhost/hookhub?sslmode=disable"))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	st := store.New(db)
	if *migrate {
		if err := st.Migrate(); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		return
	}

	rdb := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()

	q := queue.New(rdb)
	disp := dispatch.New()
	fan := dispatch.NewFanout(st, disp, q)
	worker := dispatch.NewWorker(st, disp, q)
	handlers := api.NewHandlers(st, q, disp, fan, worker, api.Config{
		ExportDir:   env("EXPORT_DIR", "/var/lib/hookhub/exports"),
		DocsDir:     env("DOCS_DIR", "/usr/share/hookhub/docs"),
		TemplateDir: env("TEMPLATE_DIR", "/var/lib/hookhub/templates"),
	})

	srv := &http.Server{
		Addr:              env("LISTEN_ADDR", ":8080"),
		Handler:           handlers.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}
