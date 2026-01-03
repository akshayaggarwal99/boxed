package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/driver"
	"github.com/akshayaggarwal99/boxed/internal/proto"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // CLI/SDK directly connecting
		}
		// In production, this should be configurable.
		// For now, allow localhost and any origin if no host is set (CLI/tests)
		return strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost")
	},
}

type Handler struct {
	driver driver.Driver
	apiKey string
}

func NewHandler(d driver.Driver, apiKey string) *Handler {
	return &Handler{
		driver: d,
		apiKey: apiKey,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/v1")

	// Apply Auth Middleware if API Key is configured
	if h.apiKey != "" {
		v1.Use(h.authMiddleware)
	}

	v1.POST("/sandbox", h.createSandbox)
	v1.POST("/sandbox/:id/exec", h.execSandbox)
	v1.DELETE("/sandbox/:id", h.stopSandbox)
	v1.GET("/sandbox", h.listSandboxes)

	// Filesystem API
	v1.GET("/sandbox/:id/files", h.listFiles)
	v1.POST("/sandbox/:id/files", h.uploadFile)
	v1.GET("/sandbox/:id/files/content", h.downloadFile)
	v1.GET("/sandbox/:id/interact", h.interactSandbox)
}

func (h *Handler) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Request().Header.Get("X-Boxed-API-Key")
		if key == "" {
			// Also support Query param for easier debugging/CLI
			key = c.QueryParam("api_key")
		}

		if h.apiKey != "" && key != h.apiKey {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing API key")
		}
		return next(c)
	}
}

func (h *Handler) listSandboxes(c echo.Context) error {
	sandboxes, err := h.driver.List(c.Request().Context(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if sandboxes == nil {
		sandboxes = []*driver.SandboxInfo{}
	}
	return c.JSON(http.StatusOK, map[string]any{"sandboxes": sandboxes})
}

type CreateSandboxRequest struct {
	Template      string                 `json:"template"`
	Timeout       int                    `json:"timeout"`
	Metadata      map[string]string      `json:"metadata"`
	NetworkPolicy driver.NetworkPolicy   `json:"network_policy"`
	Context       []driver.FileInjection `json:"context"`
}

type CreateSandboxResponse struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
}

func (h *Handler) createSandbox(c echo.Context) error {
	var req CreateSandboxRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request").SetInternal(err)
	}

	// Map template to image
	image := "python:3.10-slim" // Default fallback
	if req.Template == "python-data-science" {
		image = "boxed-python:3.9" // Assumes this image exists or will be pulled
	} else if req.Template != "" {
		// Allow direct image usage for now or map other templates
		// For MVP, just use python default if unknown, or maybe fail?
		// Let's assume template == image for advanced users if no map matches
		if strings.Contains(req.Template, ":") {
			image = req.Template
		}
	}

	cfg := driver.SandboxConfig{
		Image:         image,
		MemoryMB:      512,
		CPUCores:      1.0,
		Labels:        req.Metadata,
		Timeout:       time.Duration(req.Timeout) * time.Second,
		NetworkPolicy: req.NetworkPolicy,
		Context:       req.Context,
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	id, err := h.driver.Create(c.Request().Context(), cfg)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create sandbox: %v", err))
	}

	// Start immediately for this API model
	if err := h.driver.Start(c.Request().Context(), id); err != nil {
		// Try to verify clean up if start fails
		_ = h.driver.Stop(context.Background(), id)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start sandbox").SetInternal(err)
	}

	return c.JSON(http.StatusCreated, CreateSandboxResponse{
		SandboxID: id,
		Status:    "ready",
	})
}

type ExecRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type ExecResponse struct {
	Stdout    string                `json:"stdout"`
	Stderr    string                `json:"stderr"`
	Artifacts []proto.ArtifactEvent `json:"artifacts"`
	ExitCode  *int                  `json:"exit_code"`
}

func (h *Handler) execSandbox(c echo.Context) error {
	id := c.Param("id")
	var req ExecRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request").SetInternal(err)
	}

	// Determine command
	var cmd string
	var args []string

	switch req.Language {
	case "python":
		cmd = "python3"
		args = []string{"-c", req.Code}
	case "javascript", "node":
		cmd = "node"
		args = []string{"-e", req.Code}
	case "bash", "sh":
		cmd = "bash"
		args = []string{"-c", req.Code}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported language: "+req.Language)
	}

	// Connect to sandbox
	conn, err := h.driver.Connect(c.Request().Context(), id)
	if err != nil {
		if err == driver.ErrSandboxNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "sandbox not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to connect to sandbox").SetInternal(err)
	}
	defer conn.Close()

	// Send execution request
	rpcReq := proto.NewRequest("exec", map[string]any{
		"cmd":  cmd,
		"args": args,
	}, 1)

	reqBytes, _ := json.Marshal(rpcReq)
	if _, err := conn.Write(append(reqBytes, '\n')); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send request").SetInternal(err)
	}

	// Stream response
	// We need to read line by line until we see an "exit" event or an error response
	scanner := bufio.NewScanner(conn)

	// Increase buffer size just in case, though default is usually fine for chunks
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line

	var stdout, stderr strings.Builder
	var artifacts []proto.ArtifactEvent
	var exitCode *int

	// Set a hard timeout for the RPC loop to prevent hanging forever
	// This respects the context deadline if set by HTTP server
	done := make(chan error)
	go func() {
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Try to parse as Response first (could be initial setup response?)
			// Or Notification.
			// The protocol says: Standard Response (id:1) OR Events.
			// Wait, the Blueprint says: "Standard Response ... The Agent does not wait ... It immediately returns a stream of events."
			// BUT JSON-RPC request usually expects a response.
			// The agent implementation (which we haven't checked the logic of fully yet) likely sends events.
			// Let's assume everything coming back is a JSON object.

			var generic map[string]any
			if err := json.Unmarshal(line, &generic); err != nil {
				// Log bad JSON?
				continue
			}

			// Check if it's a response to our ID
			if idVal, ok := generic["id"]; ok && idVal != nil {
				// It's a response struct
				var resp proto.Response
				if err := json.Unmarshal(line, &resp); err != nil {
					continue
				}
				if resp.Error != nil {
					// RPC level error
					stderr.WriteString(fmt.Sprintf("\nRPC Error: %s\n", resp.Error.Message))
					// Should we stop? The exec failed to start?
					// If exec failed to start, we probably won't get events.
					break
				}
				// If result is null, it might just be ack.
				// We keep reading for events until "exit".
				continue
			}

			// It's likely a notification (StreamEvent)
			method, _ := generic["method"].(string)
			params, _ := generic["params"].(map[string]any) // generic map

			// We can re-unmarshal into specific struct or just use the map
			// Re-unmarshalling is cleaner for typed access

			switch method {
			case "stdout":
				if s, ok := params["chunk"].(string); ok {
					stdout.WriteString(s)
				}
			case "stderr":
				if s, ok := params["chunk"].(string); ok {
					stderr.WriteString(s)
				}
			case "artifact":
				// Need strict struct
				path, _ := params["path"].(string)
				mime, _ := params["mime"].(string)
				data, _ := params["data_base64"].(string)
				artifacts = append(artifacts, proto.ArtifactEvent{
					Path:       path,
					MIME:       mime,
					DataBase64: data,
				})
			case "exit":
				if c, ok := params["code"].(float64); ok { // JSON numbers are floats
					code := int(c)
					exitCode = &code
					// We are done
					done <- nil
					return
				}
			case "error":
				if msg, ok := params["message"].(string); ok {
					stderr.WriteString(fmt.Sprintf("\nRuntime Error: %s\n", msg))
				}
			}
		}
		if err := scanner.Err(); err != nil {
			done <- err
			return
		}
		// EOF without exit
		done <- io.EOF
	}()

	select {
	case <-c.Request().Context().Done():
		return echo.NewHTTPError(http.StatusRequestTimeout, "timed out")
	case err := <-done:
		if err != nil && err != io.EOF {
			return echo.NewHTTPError(http.StatusInternalServerError, "stream error").SetInternal(err)
		}
	}

	if artifacts == nil {
		artifacts = []proto.ArtifactEvent{}
	}

	result := ExecResponse{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Artifacts: artifacts,
		ExitCode:  exitCode,
	}

	return c.JSON(http.StatusOK, result)
}

func (h *Handler) stopSandbox(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}
	err := h.driver.Stop(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, driver.ErrSandboxNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "sandbox not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) listFiles(c echo.Context) error {
	id := c.Param("id")
	path := c.QueryParam("path")
	if path == "" {
		path = "/"
	}

	files, err := h.driver.ListFiles(c.Request().Context(), id, path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if files == nil {
		files = []*driver.FileEntry{}
	}
	return c.JSON(http.StatusOK, map[string]any{"files": files})
}

func (h *Handler) uploadFile(c echo.Context) error {
	id := c.Param("id")
	path := c.FormValue("path")
	if path == "" {
		path = "/uploads"
	}
	// "file" is the form field
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file required")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Append filename to path if path is a directory
	// Simplified: Assume path is destination directory?
	// Or path is full path?
	// Spec says default "/uploads".
	// Let's assume path is DIRECTORY.
	fullPath := fmt.Sprintf("%s/%s", strings.TrimSuffix(path, "/"), file.Filename)

	if err := h.driver.PutFile(c.Request().Context(), id, fullPath, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "uploaded", "path": fullPath})
}

func (h *Handler) downloadFile(c echo.Context) error {
	id := c.Param("id")
	path := c.QueryParam("path")
	if path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "path required")
	}

	content, err := h.driver.GetFile(c.Request().Context(), id, path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// Content is ReadCloser
	defer content.Close()

	return c.Stream(http.StatusOK, "application/octet-stream", content)
}
func (h *Handler) interactSandbox(c echo.Context) error {
	id := c.Param("id")

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// Connect to sandbox agent
	conn, err := h.driver.Connect(c.Request().Context(), id)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Initial Handshake: Start REPL
	// Default to bash if not specified? Or python?
	// For now, let's allow setting it via query param
	lang := c.QueryParam("lang")
	cmd := "bash"
	if lang == "python" {
		cmd = "python3"
	}

	startReq := proto.NewRequest("repl.start", map[string]any{
		"cmd": cmd,
	}, 1)
	startBytes, _ := json.Marshal(startReq)
	conn.Write(append(startBytes, '\n'))

	// Two-way Pipe
	// goroutine 1: Agent -> WebSocket
	errChan := make(chan error, 2)
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			if err := ws.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
				errChan <- err
				return
			}
		}
		errChan <- scanner.Err()
	}()

	// goroutine 2: WebSocket -> Agent
	go func() {
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}

			// If it's a raw string, we wrap it in repl.input JSON-RPC
			// This makes it easy for simple clients, but we should also allow structured JSON-RPC
			var generic map[string]any
			if err := json.Unmarshal(message, &generic); err == nil && generic["method"] != nil {
				// Already structured JSON-RPC, pass through
				conn.Write(append(message, '\n'))
			} else {
				// Raw string, wrap it
				inputReq := proto.NewRequest("repl.input", map[string]any{
					"data": string(message),
				}, nil)
				inputBytes, _ := json.Marshal(inputReq)
				conn.Write(append(inputBytes, '\n'))
			}
		}
	}()

	// Wait for any side to close
	err = <-errChan
	return err
}
