package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sandboxes",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Use global apiURL flag
		resp, err := http.Get("http://localhost:8080/v1/sandbox")
		if err != nil {
			fmt.Printf("Error connecting to server: %v\nIs the server running?\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Server returned error: %s\n", resp.Status)
			os.Exit(1)
		}

		var result struct {
			Sandboxes []struct {
				ID        string    `json:"id"`
				State     string    `json:"state"`
				CreatedAt time.Time `json:"created_at"`
				Driver    string    `json:"driver_type"`
			} `json:"sandboxes"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATE\tDRIVER\tCREATED")
		for _, s := range result.Sandboxes {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.State, s.Driver, s.CreatedAt.Format(time.RFC3339))
		}
		w.Flush()
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
