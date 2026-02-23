package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ksysoev/omnidex/pkg/cmd"
)

var (
	version = "dev"
	name    = "omnidex"
)

// main executes the entry point of the application, delegating to runApp for command execution and lifecycle management.
func main() {
	os.Exit(runApp())
}

// runApp initializes and executes the primary command for the application and manages lifecycle signals gracefully.
// It listens for termination signals (SIGINT, SIGTERM) to clean up the application state before exiting.
// Returns 1 if command execution fails due to an error; otherwise, returns 0 for successful execution.
func runApp() int {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	command := cmd.InitCommand(cmd.BuildInfo{
		Version: version,
		AppName: name,
	})

	if err := command.ExecuteContext(ctx); err != nil {
		return 1
	}

	return 0
}
