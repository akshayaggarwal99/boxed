package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings" // Added strings import
	"testing"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystem(t *testing.T) {
	// 1. Context Injection: Create with "hello.txt"
	t.Log("Testing Context Injection...")
	helloContent := "Hello from Context"
	helloBase64 := base64.StdEncoding.EncodeToString([]byte(helloContent))

	// Use map[string]any to ensure exact JSON contract matching API spec
	createPayload := map[string]any{
		"template": "python:3.10-slim",
		"timeout":  300,
		"context": []map[string]any{
			{"path": "hello.txt", "content_base64": helloBase64},
		},
	}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(BaseURL+"/sandbox", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Create failed: %s %s", resp.Status, string(b))
	}
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		SandboxID string `json:"sandbox_id"`
	}
	json.NewDecoder(resp.Body).Decode(&createResp)
	id := createResp.SandboxID
	defer func() {
		req, _ := http.NewRequest("DELETE", BaseURL+"/sandbox/"+id, nil)
		http.DefaultClient.Do(req)
	}()

	// Wait for start
	time.Sleep(2 * time.Second)

	// Verify Context Injection: Read "hello.txt"
	t.Log("Verifying Context Injection...")
	resp, err = http.Get(fmt.Sprintf("%s/sandbox/%s/files/content?path=hello.txt", BaseURL, id))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	content, _ := io.ReadAll(resp.Body)
	assert.Equal(t, helloContent, string(content))

	// 2. Upload File (PUT)
	t.Log("Testing File Upload...")
	uploadContent := "Uploaded Content"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("path", "/workspace") // Use explicit path
	fw, _ := w.CreateFormFile("file", "upload.txt")
	fw.Write([]byte(uploadContent))
	w.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/sandbox/%s/files", BaseURL, id), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify Upload: Execute code to cat the file
	execPayload := map[string]string{
		"code":     "print(open('upload.txt').read(), end='')",
		"language": "python", // Explicit language
	}
	execBody, _ := json.Marshal(execPayload)
	resp, err = http.Post(fmt.Sprintf("%s/sandbox/%s/exec", BaseURL, id), "application/json", bytes.NewReader(execBody))
	require.NoError(t, err)
	// Check status code
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Exec failed: %s %s", resp.Status, string(b))
	}
	var execResp struct {
		Stdout string `json:"stdout"`
	}
	json.NewDecoder(resp.Body).Decode(&execResp)
	// Normalize stdout
	assert.Equal(t, uploadContent, strings.TrimSpace(execResp.Stdout))

	// 3. List Files
	t.Log("Testing List Files...")
	resp, err = http.Get(fmt.Sprintf("%s/sandbox/%s/files?path=/", BaseURL, id))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var files []driver.FileEntry
	json.NewDecoder(resp.Body).Decode(&files)

	// Should see hello.txt and upload.txt
	foundHello := false
	foundUpload := false
	for _, f := range files {
		if filepath.Base(f.Name) == "hello.txt" {
			foundHello = true
		}
		if filepath.Base(f.Name) == "upload.txt" {
			foundUpload = true
		}
	}
	assert.True(t, foundHello, "hello.txt found")
	assert.True(t, foundUpload, "upload.txt found")

	// 4. Generated Artifacts (Code -> File -> Download)
	t.Log("Testing Artifact Generation...")
	// Create a file via python. Ensure directory exists.
	genPayload := map[string]string{
		"code":     "f = open('/workspace/plot.png', 'w'); f.write('fake png content'); f.close()",
		"language": "python",
	}
	genBody, _ := json.Marshal(genPayload)
	resp, err = http.Post(fmt.Sprintf("%s/sandbox/%s/exec", BaseURL, id), "application/json", bytes.NewReader(genBody))
	require.NoError(t, err)
	// Check response contents
	var genResp struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode *int   `json:"exit_code"`
	}
	json.NewDecoder(resp.Body).Decode(&genResp)
	if genResp.ExitCode == nil || *genResp.ExitCode != 0 {
		t.Fatalf("Artifact gen failed. Exit: %v, Stderr: %s", genResp.ExitCode, genResp.Stderr)
	}

	// Verify it exists in List
	resp, err = http.Get(fmt.Sprintf("%s/sandbox/%s/files?path=/workspace", BaseURL, id))
	require.NoError(t, err)
	json.NewDecoder(resp.Body).Decode(&files)

	foundPlot := false
	for _, f := range files {
		if f.Name == "plot.png" {
			foundPlot = true
			break
		}
	}
	assert.True(t, foundPlot, "plot.png found in /workspace")

	// Download it
	resp, err = http.Get(fmt.Sprintf("%s/sandbox/%s/files/content?path=/workspace/plot.png", BaseURL, id))
	require.NoError(t, err)
	content, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "fake png content", string(content))
}
