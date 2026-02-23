package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ksysoev/omnidex/pkg/api"
	"github.com/spf13/viper"
)

type appConfig struct {
	Storage StorageConfig `mapstructure:"storage"`
	Search  SearchConfig  `mapstructure:"search"`
	API     api.Config    `mapstructure:"api"`
}

// StorageConfig holds configuration for document storage.
type StorageConfig struct {
	Path string `mapstructure:"path"`
}

// SearchConfig holds configuration for the search engine.
type SearchConfig struct {
	IndexPath string `mapstructure:"index_path"`
}

// loadConfig loads the application configuration from the specified file path and environment variables.
// It uses the provided args structure to determine the configuration path.
// The function returns a pointer to the appConfig structure and an error if something goes wrong.
func loadConfig(flags *cmdFlags) (*appConfig, error) {
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())

	if flags.ConfigPath != "" {
		v.SetConfigFile(flags.ConfigPath)

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg appConfig

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	slog.Debug("Config loaded", slog.Any("config", cfg))

	return &cfg, nil
}
