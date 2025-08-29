package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/manto/manto-web/internal/config"
	"github.com/manto/manto-web/internal/handlers"
	"github.com/manto/manto-web/internal/middleware/security"
	"github.com/manto/manto-web/internal/services"
)

//go:embed static/*
var embeddedStatic embed.FS

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	port := cfg.Server.Port

	anthropicService := services.NewAnthropicService(cfg)
	apiHandlers := handlers.NewAPIHandlers(cfg, anthropicService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(security.SecurityHeaders(cfg))

	r.Get("/config.js", apiHandlers.ConfigHandler)
	r.Get("/api/models", apiHandlers.ModelsHandler)
	r.Post("/api/messages", apiHandlers.MessagesHandler)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	fileServer := http.FileServer(http.FS(sub))
	r.Handle("/*", fileServer)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Manto starting on port %d (%s)", port, config.GetEnvironment())
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
