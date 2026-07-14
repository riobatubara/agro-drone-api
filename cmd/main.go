package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agro-drone-api/generated"
	"agro-drone-api/handler"
	"agro-drone-api/repository"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Initialize the generated OpenAPI handlers
	var server generated.ServerInterface = newServer()
	generated.RegisterHandlers(e, server)

	// Set up background listener for graceful shutdown (SIGINT/SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := "1323"

	// Start the server inside a background goroutine thread
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("shutting down the server due to error: %v", err)
		}
	}()

	// Pause execution here until an OS close signal is received
	<-ctx.Done()
	e.Logger.Info("termination signal received: starting safe shutdown...")

	// Allow active connection tasks up to 10 seconds to finish before forcing a exit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatalf("graceful shutdown failed: %v", err)
	}
	e.Logger.Info("server shutdown executed securely")
}

func newServer() *handler.Server {
	dbDsn := os.Getenv("DATABASE_URL")
	if dbDsn == "" {
		dbDsn = "postgres://postgres:postgres@db:5432/database?sslmode=disable"
	}
	var repo repository.RepositoryInterface = repository.NewRepository(repository.NewRepositoryOptions{
		Dsn: dbDsn,
	})
	opts := handler.NewServerOptions{
		Repository: repo,
	}
	return handler.NewServer(opts)
}
