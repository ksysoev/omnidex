package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksysoev/omnidex/pkg/api"
	"github.com/ksysoev/omnidex/pkg/prov/someapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	const validConfig = `
api:
  listen: ":8082"
redis:
  addr: "localhost:6379"
  password: "testpassword"
provider:
  some_api:
    base_url: "https://api.example.com"
`

	tests := []struct {
		envVars      map[string]string
		expectConfig *appConfig
		name         string
		configData   string
		expectError  bool
	}{
		{
			name:        "valid config file",
			envVars:     nil,
			expectError: false,
			configData:  validConfig,
			expectConfig: &appConfig{
				API: api.Config{
					Listen: ":8082",
				},
				Redis: RedisConfig{
					Addr:     "localhost:6379",
					Password: "testpassword",
				},
				Provider: Provider{
					SomeAPI: someapi.Config{
						BaseURL: "https://api.example.com",
					},
				},
			},
		},
		{
			name:        "missing config file",
			envVars:     nil,
			expectError: true,
		},
		{
			name:        "unparseable config file",
			envVars:     nil,
			expectError: true,
			configData:  `invalid yaml`,
		},
		{
			name: "valid config with environment overrides",
			envVars: map[string]string{
				"API_LISTEN":                 ":8083",
				"PROVIDER_SOME_API_BASE_URL": "https://test.com",
			},
			expectError: false,
			configData:  validConfig,
			expectConfig: &appConfig{
				API: api.Config{
					Listen: ":8083",
				},
				Redis: RedisConfig{
					Addr:     "localhost:6379",
					Password: "testpassword",
				},
				Provider: Provider{
					SomeAPI: someapi.Config{
						BaseURL: "https://test.com",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if tt.configData != "" {
				err := os.WriteFile(configPath, []byte(tt.configData), 0o600)

				require.NoError(t, err)
			}

			// Set up environment variables
			if tt.envVars != nil {
				for key, value := range tt.envVars {
					_ = os.Setenv(key, value)

					t.Cleanup(func() {
						_ = os.Unsetenv(key)
					})
				}
			}

			arg := &cmdFlags{
				ConfigPath: configPath,
			}

			cfg, err := loadConfig(arg)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectConfig, cfg)
			}
		})
	}
}
