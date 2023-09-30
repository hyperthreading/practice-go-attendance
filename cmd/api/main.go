package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hyperthreading/go-attendance/internal/api"
)

func main() {
	r := api.New()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		// listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	<-ctx.Done()
	log.Println("Timeout of 5 seconds.")

	log.Println("Server stopped")
}
