package scenarios

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Endpoint string
	APIKey   string
	HTTP     *http.Client
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Client) do(method, path string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Endpoint+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("X-Boxed-API-Key", c.APIKey)
	}
	return c.http().Do(req)
}

type createResp struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
}

func (c Client) Create(template string) (string, error) {
	body := map[string]any{"template": template, "timeout": 300}
	resp, err := c.do("POST", "/v1/sandbox", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create %d: %s", resp.StatusCode, string(b))
	}
	var out createResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.SandboxID, nil
}

type execResp struct {
	Stdout    string          `json:"stdout"`
	Stderr    string          `json:"stderr"`
	ExitCode  *int            `json:"exit_code"`
	Artifacts json.RawMessage `json:"artifacts"`
}

func (c Client) Exec(id, language, code string) (execResp, error) {
	body := map[string]any{"language": language, "code": code}
	resp, err := c.do("POST", "/v1/sandbox/"+id+"/exec", body)
	if err != nil {
		return execResp{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return execResp{}, fmt.Errorf("exec %d: %s", resp.StatusCode, string(b))
	}
	var out execResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return execResp{}, err
	}
	return out, nil
}

func (c Client) Destroy(id string) error {
	resp, err := c.do("DELETE", "/v1/sandbox/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("destroy %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }
