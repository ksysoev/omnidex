package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

const healthCheckTimeout = 5 * time.Second

// newHealthCmd creates a cobra command that checks the health of a running omnidex instance.
// It performs an HTTP GET request to the /livez endpoint and reports whether the server is healthy.
func newHealthCmd() *cobra.Command {
	var url string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check the health of a running omnidex instance",
		Long:  "Perform a health check against a running omnidex instance by querying the /livez endpoint.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHealthCheck(cmd.Context(), url)
		},
	}

	cmd.Flags().StringVar(&url, "url", "http://localhost:8080", "base URL of the omnidex instance")

	return cmd
}

// runHealthCheck performs an HTTP GET to the /livez endpoint at the given base URL.
// It returns nil if the server responds with HTTP 200, or an error otherwise.
func runHealthCheck(ctx context.Context, baseURL string) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	endpoint := baseURL + "/livez"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // URL is user-provided via CLI flag, not tainted input
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	fmt.Println("ok") //nolint:forbidigo // CLI output is intentional

	return nil
}
