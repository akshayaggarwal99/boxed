package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	template string
	timeout  int
)

var runCmd = &cobra.Command{
	Use:   "run [code]",
	Short: "Run code in a ephemeral sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		code := args[0]

		// 1. Create Sandbox
		createPayload := map[string]any{
			"template": template,
			"timeout":  timeout,
		}
		body, _ := json.Marshal(createPayload)

		resp, err := http.Post("http://localhost:8080/v1/sandbox", "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Failed to connect: %v\nIs the server running?\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			fmt.Printf("Create failed: %s\n", resp.Status)
			io.Copy(os.Stderr, resp.Body)
			os.Exit(1)
		}

		var createResp struct {
			ID string `json:"sandbox_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
			fmt.Printf("Bad response: %v\n", err)
			os.Exit(1)
		}
		id := createResp.ID
		fmt.Printf("📦 Sandbox %s created\n", id)

		// 2. Execute Code
		execPayload := map[string]string{
			"code":     code,
			"language": "python", // MVP: Assume python based on default template
		}
		body, _ = json.Marshal(execPayload)
		resp, err = http.Post(fmt.Sprintf("http://localhost:8080/v1/sandbox/%s/exec", id), "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Exec failed: %v\n", err)
			cleanup(id)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var execResp struct {
			Stdout    string `json:"stdout"`
			Stderr    string `json:"stderr"`
			ExitCode  *int   `json:"exit_code"`
			Artifacts []struct {
				Path       string `json:"path"`
				Mime       string `json:"mime"`
				DataBase64 string `json:"data_base64"`
			} `json:"artifacts"`
		}
		json.NewDecoder(resp.Body).Decode(&execResp)

		fmt.Print(execResp.Stdout)
		if execResp.Stderr != "" {
			fmt.Fprint(os.Stderr, execResp.Stderr)
		}

		// Handle artifacts
		if len(execResp.Artifacts) > 0 {
			os.Mkdir("artifacts", 0755)
			fmt.Println("\n📂 Artifacts:")
			for _, a := range execResp.Artifacts {
				data, err := base64.StdEncoding.DecodeString(a.DataBase64)
				if err != nil {
					fmt.Printf("  - Failed to decode %s: %v\n", a.Path, err)
					continue
				}
				// Save locally
				localPath := filepath.Join("artifacts", filepath.Base(a.Path))
				err = os.WriteFile(localPath, data, 0644)
				if err != nil {
					fmt.Printf("  - Failed to write %s: %v\n", localPath, err)
				} else {
					fmt.Printf("  - Saved: %s (%s)\n", localPath, a.Mime)
				}
			}
		}

		// 3. Cleanup
		cleanup(id)
		fmt.Printf("\n♻️  Sandbox destroyed\n")
	},
}

func cleanup(id string) {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:8080/v1/sandbox/%s", id), nil)
	http.DefaultClient.Do(req)
}

func init() {
	runCmd.Flags().StringVarP(&template, "template", "t", "python:3.10-slim", "Sandbox template image")
	runCmd.Flags().IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	RootCmd.AddCommand(runCmd)
}
