package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxLifecycle(t *testing.T) {
	// 1. Create Sandbox
	t.Log("Creating sandbox...")
	createPayload := map[string]any{
		"template": "python:3.10-slim",
		"timeout":  300,
	}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(BaseURL+"/sandbox", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		SandboxID string `json:"sandbox_id"`
		Status    string `json:"status"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(t, err)
	sandboxID := createResp.SandboxID
	require.NotEmpty(t, sandboxID)

	defer func() {
		// Cleanup
		req, _ := http.NewRequest("DELETE", BaseURL+"/sandbox/"+sandboxID, nil)
		http.DefaultClient.Do(req)
	}()

	// 2. Execute Code
	t.Log("Executing code...")
	execPayload := map[string]string{
		"language": "python",
		"code":     "print('Lifecycle Test Success')",
	}
	body, _ = json.Marshal(execPayload)

	// Retry loop for cold start
	var execResp struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode *int   `json:"exit_code"`
	}

	for i := 0; i < 5; i++ {
		resp, err = http.Post(fmt.Sprintf("%s/sandbox/%s/exec", BaseURL, sandboxID), "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(&execResp)
			break
		}
		time.Sleep(1 * time.Second)
	}

	assert.Contains(t, execResp.Stdout, "Lifecycle Test Success")

	// 3. List Sandboxes
	resp, err = http.Get(BaseURL + "/sandbox")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Sandboxes []struct {
			ID string `json:"id"`
		} `json:"sandboxes"`
	}
	json.NewDecoder(resp.Body).Decode(&listResp)

	found := false
	for _, s := range listResp.Sandboxes {
		if s.ID == sandboxID {
			found = true
			break
		}
	}
	assert.True(t, found, "Sandbox should be listed")
}
