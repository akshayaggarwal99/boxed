// Package main is the entry point for the Boxed Control Plane server.
//
// Boxed is a distributed system for provisioning, managing, and communicating
// with ephemeral execution environments for AI agents and developers.
//
// Usage:
//
//	boxed-server [flags]
//
// Flags:
//
//	-c, --config string   Path to config file (default: boxed.yaml)
//	-p, --port int        HTTP server port (default: 8080)
//	-d, --driver string   Backend driver: docker, firecracker (default: docker)
//	-v, --verbose         Enable debug logging
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/api"
	"github.com/akshayaggarwal99/boxed/internal/driver"

	// Register docker driver
	_ "github.com/akshayaggarwal99/boxed/internal/driver/docker"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version information (set via ldflags at build time)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Configure structured JSON logging
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Use pretty console output for development
	if os.Getenv("BOXED_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	}

	log.Info().
		Str("version", Version).
		Str("commit", GitCommit).
		Str("built", BuildDate).
		Msg("🗳️  Boxed Control Plane starting")

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("Shutdown signal received")
		cancel()
	}()

	// Initialize configuration
	// For MVP, we stick to defaults or env vars handled by driver New()

	// Create Docker driver
	d, err := driver.NewDriver("docker", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize docker driver")
	}
	defer d.Close()

	// Verify driver health
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 5*time.Second)
	if err := d.Healthy(ctxTimeout); err != nil {
		log.Fatal().Err(err).Msg("Driver health check failed")
	}
	cancelTimeout()

	// Initialize API
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Add middleware if needed (Logger, Recover)
	// e.Use(middleware.Logger())
	// e.Use(middleware.Recover())

	apiKey := os.Getenv("BOXED_API_KEY")
	h := api.NewHandler(d, apiKey)
	h.RegisterRoutes(e)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		port := "8080"
		if p := os.Getenv("PORT"); p != "" {
			port = p
		}
		log.Info().Str("port", port).Msg("🚀 Server listening")
		serverErr <- e.Start(":" + port)
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("Server startup failed")
	}
}
