package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ksysoev/omnidex/pkg/api"
	"github.com/ksysoev/omnidex/pkg/prov/someapi"
	"github.com/spf13/viper"
)

type appConfig struct {
	API      api.Config  `mapstructure:"api"`
	Redis    RedisConfig `mapstructure:"redis"`
	Provider Provider    `mapstructure:"provider"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password" json:"-"`
}

type Provider struct {
	SomeAPI someapi.Config `mapstructure:"some_api"`
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
