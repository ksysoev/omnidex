package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type BuildInfo struct {
	Version string
	AppName string
}

type cmdFlags struct {
	version    string
	appName    string
	ConfigPath string `mapstructure:"config"`
	LogLevel   string `mapstructure:"log_level"`
	TextFormat bool   `mapstructure:"log_text"`
}

// InitCommand initializes the root command of the CLI application with its subcommands and flags.
func InitCommand(build BuildInfo) cobra.Command {
	flags := cmdFlags{
		version: build.Version,
		appName: build.AppName,
	}

	cmd := cobra.Command{
		Use:   flags.appName,
		Short: "Centralized documentation portal for your repos",
		Long:  "Omnidex is a centralized documentation portal that aggregates and serves documentation from your repositories.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunCommand(cmd.Context(), &flags)
		},
	}

	cmd.PersistentFlags().StringVar(&flags.LogLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().BoolVar(&flags.TextFormat, "log-text", true, "log in text format, otherwise JSON")
	cmd.PersistentFlags().StringVar(&flags.ConfigPath, "config", "runtime/config.yml", "path to the configuration file")

	for _, name := range []string{"log_level", "log_text"} {
		if err := viper.BindEnv(name); err != nil {
			slog.Error("failed to bind env var", "name", name, "error", err)
		}
	}

	viper.AutomaticEnv()

	if err := viper.Unmarshal(&flags); err != nil {
		slog.Error("failed to unmarshal env vars", "error", err)
	}

	return cmd
}
