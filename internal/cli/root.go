package cli

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "glitch [db.json | api.yaml]",
	Short: "A flaky API server for frontend development",
	Long: `Glitch is a JSON API server, OpenAPI mock server, and reverse proxy with built-in chaos engineering.
It serves a REST API, mocks OpenAPI responses, or proxies to a real backend — with configurable latency
injection, random failures, and other chaos features — perfect for
testing how your frontend handles unreliable backends.`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runServe,
	Version: Version,
}

func init() {
	f := rootCmd.Flags()

	f.IntP("port", "p", 3000, "server port")
	f.String("host", "localhost", "server host")
	f.String("latency", "", "inject latency (e.g. \"2s\", \"500ms-3s\", \"normal:200ms,2s\")")
	f.String("fail-rate", "", "percentage of requests that fail (e.g. \"20\" or \"20%\")")
	f.StringSlice("status", nil, "error status codes with rates (e.g. \"500:10\", \"429:5\")")
	f.String("profile", "", "load a chaos profile (e.g. \"mobile\", \"degraded\")")
	f.BoolP("verbose", "v", false, "enable verbose request logging")
	f.Bool("read-only", false, "disable all write operations (POST, PUT, PATCH, DELETE)")
	f.String("proxy", "", "reverse proxy to target URL (e.g. \"http://api.example.com\")")
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
