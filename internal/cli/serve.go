package cli

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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	port       string
	driverName string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Boxed Control Plane server",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "HTTP server port")
	serveCmd.Flags().StringVarP(&driverName, "driver", "d", "docker", "Backend driver: docker, firecracker")
	serveCmd.Flags().StringVar(&apiKey, "api-key", os.Getenv("BOXED_API_KEY"), "API Key for authentication")
	RootCmd.AddCommand(serveCmd)
}

func runServer() {
	log.Info().Str("driver", driverName).Str("port", port).Msg("🗳️  Starting Boxed Server")

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

	// Init Driver
	d, err := driver.NewDriver(driverName, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize driver")
	}
	defer d.Close()

	// Health Check
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 5*time.Second)
	if err := d.Healthy(ctxTimeout); err != nil {
		log.Fatal().Err(err).Msg("Driver health check failed")
	}
	cancelTimeout()

	// Init API
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	h := api.NewHandler(d, apiKey)
	h.RegisterRoutes(e)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("port", port).Msg("🚀 Server listening")
		serverErr <- e.Start(":" + port)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("Server startup failed")
	}
}
