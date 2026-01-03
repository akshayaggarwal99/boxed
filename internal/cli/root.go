package cli

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	jsonLog bool
	apiKey  string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "boxed",
	Short: "Sovereign Code Execution Engine",
	Long: `Boxed is a high-performance, secure sandbox orchestration platform 
designed for running untrusted AI-generated code.

It provides both a server for managing sandboxes (Docker/Firecracker)
and client utilities for interacting with the API.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure logging
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

		if !jsonLog {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}

		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	RootCmd.PersistentFlags().BoolVar(&jsonLog, "json-log", false, "Output logs in JSON format")
	RootCmd.PersistentFlags().StringVar(&apiKey, "api-key", os.Getenv("BOXED_API_KEY"), "API Key for authentication")
}
