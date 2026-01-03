package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/driver"
	"github.com/akshayaggarwal99/boxed/internal/proto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestREPLStickySessions(t *testing.T) {
	ctx := context.Background()

	// 1. Create Sandbox
	id, err := testDriver.Create(ctx, driver.SandboxConfig{
		Image:   "python:3.10-slim",
		Timeout: 60 * time.Second,
	})
	require.NoError(t, err)
	defer testDriver.Stop(ctx, id)

	err = testDriver.Start(ctx, id)
	require.NoError(t, err)

	// 2. Connect via WebSocket
	u, err := url.Parse(BaseURL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = fmt.Sprintf("%s/sandbox/%s/interact", u.Path, id)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	// 3. Send command (wrapped in JSON-RPC)
	// We use repl.input to send stdin
	inputReq := proto.NewRequest("repl.input", map[string]any{
		"data": "echo 'boxed-session-id-123'\n",
	}, nil)
	inputBytes, _ := json.Marshal(inputReq)
	err = c.WriteMessage(websocket.TextMessage, inputBytes)
	require.NoError(t, err)

	// 4. Read output and verify
	found := false
	timeout := time.After(10 * time.Second)
	for !found {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for REPL output")
		default:
			_, message, err := c.ReadMessage()
			require.NoError(t, err)

			var event struct {
				Method string `json:"method"`
				Params struct {
					Chunk string `json:"chunk"`
				} `json:"params"`
			}
			if err := json.Unmarshal(message, &event); err == nil {
				if event.Method == "stdout" && strings.Contains(event.Params.Chunk, "boxed-session-id-123") {
					found = true
				}
			}
		}
	}
	require.True(t, found)
}
