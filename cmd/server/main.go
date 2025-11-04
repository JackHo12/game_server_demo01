package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/matchmaker-lite/internal/api"
	"github.com/yourname/matchmaker-lite/internal/metrics"
	"github.com/yourname/matchmaker-lite/internal/match"
	"github.com/yourname/matchmaker-lite/internal/store"
	"github.com/yourname/matchmaker-lite/internal/ws"
)

func main() {
	addr := getEnv("HTTP_ADDR", ":8080")
	redisURL := getEnv("REDIS_ADDR", "localhost:6379")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis store
	st := store.NewRedisStore(redisURL, "")
	defer st.Close()

	// Event hub for server-sent events to WS clients
	hub := ws.NewHub()
	go hub.Run()

	// Matchmaker
	mm := match.NewMatchmaker(st, hub)
	go mm.Run(ctx)

	r := api.NewRouter(st, hub, mm)

	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		log.Printf("http listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// metrics
	metrics.Init()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctxShut, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShut()
	_ = srv.Shutdown(ctxShut)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}