package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name    string
		flags   cmdFlags
		wantErr bool
	}{
		{
			name: "Valid log level with text format",
			flags: cmdFlags{
				LogLevel:   "info",
				TextFormat: true,
				version:    "1.0.0",
				appName:    "test-app",
			},
			wantErr: false,
		},
		{
			name: "Valid log level with JSON format",
			flags: cmdFlags{
				LogLevel:   "debug",
				TextFormat: false,
				version:    "1.0.0",
				appName:    "test-app",
			},
			wantErr: false,
		},
		{
			name: "Invalid log level",
			flags: cmdFlags{
				LogLevel:   "invalid",
				TextFormat: true,
				version:    "1.0.0",
				appName:    "test-app",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := initLogger(&tt.flags)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
