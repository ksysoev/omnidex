package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidConfig(t *testing.T) {
	cfg := Config{Listen: ":8080", APIKeys: []string{"key1"}}
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api, err := New(cfg, svc, views)

	require.NoError(t, err)
	assert.NotNil(t, api)
}

func TestNew_EmptyListen(t *testing.T) {
	cfg := Config{Listen: ""}
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	_, err := New(cfg, svc, views)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "listen address must be specified")
}

func TestRun_GracefulShutdown(t *testing.T) {
	cfg := Config{Listen: "127.0.0.1:0", APIKeys: []string{"key1"}}
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api, err := New(cfg, svc, views)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = api.Run(ctx)
	assert.NoError(t, err)
}
