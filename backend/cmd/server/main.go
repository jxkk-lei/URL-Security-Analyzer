package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-security-analyzer/backend/internal/config"
	"url-security-analyzer/backend/internal/database"
	"url-security-analyzer/backend/internal/handler"
	"url-security-analyzer/backend/internal/repository"
	"url-security-analyzer/backend/internal/server"
	"url-security-analyzer/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Server stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	dbCtx, cancelDB := context.WithTimeout(
		signalCtx,
		10*time.Second,
	)

	db, err := database.NewPool(dbCtx, cfg.DatabaseURL)
	cancelDB()

	if err != nil {
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}

	log.Println("Database migrations applied successfully")

	checksRepository := repository.NewChecksRepository(db)
	checksService := service.NewChecksService(checksRepository)

	checksHandler := &handler.ChecksHandler{
		Service: checksService,
	}

	healthHandler := &handler.HealthHandler{
		DB: db,
	}

	httpServer := server.New(
		cfg.ServerAddr,
		checksHandler,
		healthHandler,
	)

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("URL Security Analyzer API started on %s", cfg.ServerAddr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}

		return nil

	case <-signalCtx.Done():
		log.Println("Shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("Force close failed: %v", closeErr)
		}
	}

	err = <-serverErrors
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server failed during shutdown: %w", err)
	}

	log.Println("HTTP server stopped")

	return nil
}
