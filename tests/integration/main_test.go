package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/api"
	"github.com/akshayaggarwal99/boxed/internal/driver"

	// Register drivers
	_ "github.com/akshayaggarwal99/boxed/internal/driver/docker"
	"github.com/labstack/echo/v4"
)

var testDriver driver.Driver

const (
	ServerPort = "8081" // Use different port than default to avoid conflict
	BaseURL    = "http://localhost:" + ServerPort + "/v1"
)

func TestMain(m *testing.M) {
	// Fix WD to project root so driver can find agent binary
	os.Chdir("../..")

	// Setup: Start Server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Use Docker driver for integration tests
	// We need to ensure we can connect to Docker
	var err error
	testDriver, err = driver.NewDriver("docker", nil)
	if err != nil {
		fmt.Printf("Failed to init driver: %v\n", err)
		os.Exit(1)
	}

	// Health check
	if err := testDriver.Healthy(context.Background()); err != nil {
		fmt.Printf(" Docker unreachable, skipping integration tests: %v\n", err)
		os.Exit(0)
	}

	h := api.NewHandler(testDriver, "")
	h.RegisterRoutes(e)

	go func() {
		if err := e.Start(":" + ServerPort); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for server to be ready
	waitForServer()

	// Run Tests
	code := m.Run()

	// Teardown
	testDriver.Close()
	e.Shutdown(context.Background())
	os.Exit(code)
}

func waitForServer() {
	for i := 0; i < 10; i++ {
		resp, err := http.Get(BaseURL + "/sandbox")
		if err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("Timeout waiting for test server")
	os.Exit(1)
}
