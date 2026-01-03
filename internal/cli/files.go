package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "fs",
	Short: "Manage files in a sandbox",
}

var lsCmd = &cobra.Command{
	Use:   "ls [sandbox-id] [path]",
	Short: "List files in directory",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		path := "/"

		// Check for ID:path syntax
		if parts := splitRemote(id); parts != nil {
			id = parts[0]
			path = parts[1]
		} else if len(args) > 1 {
			path = args[1]
		}

		resp, err := http.Get(fmt.Sprintf("http://localhost:8080/v1/sandbox/%s/files?path=%s", id, path))
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error: %s\n", resp.Status)
			io.Copy(os.Stderr, resp.Body)
			os.Exit(1)
		}

		var files []struct {
			Name         string    `json:"name"`
			Size         int64     `json:"size"`
			IsDir        bool      `json:"is_dir"`
			LastModified time.Time `json:"last_modified"`
		}
		json.NewDecoder(resp.Body).Decode(&files)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "MODE\tSIZE\tUPDATED\tNAME")
		for _, f := range files {
			mode := "-rw-r--r--"
			if f.IsDir {
				mode = "drwxr-xr-x"
			}
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", mode, f.Size, f.LastModified.Format(time.RFC822), f.Name)
		}
		w.Flush()
	},
}

var putCmd = &cobra.Command{
	Use:   "cp [local-path] [sandbox-id]:[remote-path]",
	Short: "Upload file to sandbox",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse Local
		localPath := args[0]

		// Parse Remote
		// Format: ID:/path/to/file
		// Or assume args[1] is ID and we need a flag?
		// Let's stick to standard `cp` syntax: `boxed fs cp ./data.csv <id>:/workspace/`

		// Wait, args[1] is the target.
		// "sandbox-id:path"
		// Splitting logic:
		remote := args[1]
		parts := splitRemote(remote)
		if parts == nil {
			fmt.Println("Invalid remote format. Use ID:/path/to/dest")
			os.Exit(1)
		}
		id := parts[0]
		remotePath := parts[1]

		// Upload
		file, err := os.Open(localPath)
		if err != nil {
			fmt.Printf("Failed to open local file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		// Prepare Multipart
		r, w := io.Pipe()
		m := multipart.NewWriter(w)

		go func() {
			defer w.Close()
			defer m.Close()

			// We need to set "path" field?
			// The handler expects form value "path" to be definition of destination directory?
			// Or full path?
			// Handler: `fullPath := fmt.Sprintf("%s/%s", strings.TrimSuffix(path, "/"), file.Filename)`
			// So "path" param should be the DIRECTORY.

			// If remotePath is "/foo/bar.txt", handler logic implies we pass "/foo" and filename "bar.txt".
			// Let's just pass filepath.Dir(remotePath) ?
			// Actually let's assume remotePath is the DIRECTORY where we want to put it.
			// But cp usually allows renaming.
			// Let's rely on standard practice: If remotePath ends in /, it's a dir.
			// If not, it's a file rename?
			// Handler logic currently forces appending filename. So we CANNOT rename.
			// MVP: remotePath IS the directory.

			m.WriteField("path", remotePath)

			part, err := m.CreateFormFile("file", filepath.Base(localPath))
			if err != nil {
				return
			}
			io.Copy(part, file)
		}()

		req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:8080/v1/sandbox/%s/files", id), r)
		req.Header.Set("Content-Type", m.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Upload failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error: %s\n", resp.Status)
			io.Copy(os.Stderr, resp.Body)
			os.Exit(1)
		}
		fmt.Printf("Uploaded to %s:%s\n", id, remotePath)
	},
}

var getCmd = &cobra.Command{
	Use:   "cat [sandbox-id] [path]",
	Short: "Download/Cat file content",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		path := ""

		// Check for ID:path syntax
		if parts := splitRemote(id); parts != nil {
			id = parts[0]
			path = parts[1]
		} else if len(args) > 1 {
			path = args[1]
		}

		if path == "" {
			fmt.Println("Path is required. Use ID:path or pass path as second argument")
			os.Exit(1)
		}

		resp, err := http.Get(fmt.Sprintf("http://localhost:8080/v1/sandbox/%s/files/content?path=%s", id, path))
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error: %s\n", resp.Status)
			io.Copy(os.Stderr, resp.Body)
			os.Exit(1)
		}

		io.Copy(os.Stdout, resp.Body)
	},
}

func init() {
	filesCmd.AddCommand(lsCmd)
	filesCmd.AddCommand(putCmd)
	filesCmd.AddCommand(getCmd)
	RootCmd.AddCommand(filesCmd)
}

func splitRemote(s string) []string {
	// Simple split by first colon
	for i, c := range s {
		if c == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
