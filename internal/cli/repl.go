package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:   "repl [sandbox-id]",
	Short: "Start an interactive session in a sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		// Determine language (optional)
		lang, _ := cmd.Flags().GetString("lang")

		u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: fmt.Sprintf("/v1/sandbox/%s/interact", id)}
		if lang != "" {
			u.RawQuery = "lang=" + lang
		}

		fmt.Printf("Connecting to %s...\n", u.String())

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			fmt.Printf("Dial failed: %v\n", err)
			os.Exit(1)
		}
		defer c.Close()

		fmt.Println("Connected! Type your commands below. CTRL+C to exit.")

		done := make(chan struct{})
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)

		// goroutine: WS -> Local Stdout
		go func() {
			defer close(done)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					fmt.Printf("\nConnection closed: %v\n", err)
					return
				}

				// Try to parse as JSON-RPC Event
				var event struct {
					Method string `json:"method"`
					Params struct {
						Chunk   string `json:"chunk"`
						Message string `json:"message"`
						Code    int    `json:"code"`
					} `json:"params"`
				}

				if err := json.Unmarshal(message, &event); err == nil {
					switch event.Method {
					case "stdout", "stderr":
						fmt.Print(event.Params.Chunk)
					case "error":
						fmt.Printf("\n[Error] %s\n", event.Params.Message)
					case "exit":
						fmt.Printf("\n[Process Exited with code %d]\n", event.Params.Code)
						return
					}
				} else {
					// Fallback for non-JSON or other messages
					fmt.Print(string(message))
				}
			}
		}()

		// goroutine: Local Stdin -> WS
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					if err != io.EOF {
						fmt.Printf("\nRead error: %v\n", err)
					}
					return
				}
				if n > 0 {
					err := c.WriteMessage(websocket.TextMessage, buf[:n])
					if err != nil {
						fmt.Printf("\nWrite error: %v\n", err)
						return
					}
				}
			}
		}()

		select {
		case <-done:
			return
		case <-interrupt:
			fmt.Println("Interrupt received, closing...")
			// Cleanly close the connection by sending a close message and then
			// waiting (with a timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				fmt.Printf("Write close error: %v\n", err)
				return
			}
			select {
			case <-done:
			case <-time.After(1 * time.Second):
			}
			return
		}
	},
}

func init() {
	replCmd.Flags().StringP("lang", "l", "bash", "Language/Shell (bash, python)")
	RootCmd.AddCommand(replCmd)
}
