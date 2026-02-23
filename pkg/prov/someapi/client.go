// Package someapi provides a client for interacting with the SomeAPI service.
package someapi

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const defaultTimeout = 5 * time.Second

// Config holds configuration for the APIClient.
type Config struct {
	BaseURL string `mapstructure:"base_url"`
}

// APIClient is a client for communicating with the SomeAPI service.
type APIClient struct {
	cli *http.Client
	cfg Config
}

// New creates a new APIClient with the provided configuration.
func New(cfg Config) *APIClient {
	return &APIClient{
		cfg: cfg,
		cli: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CheckHealth checks the health status of the SomeAPI service.
func (a *APIClient) CheckHealth(ctx context.Context) error {
	resp, err := a.cli.Get(a.cfg.BaseURL + "/livez")
	if err != nil {
		return fmt.Errorf("fail to check health status for someapi: %w", err)
	}

	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status of someapi is unhealthy")
	}

	return nil
}
