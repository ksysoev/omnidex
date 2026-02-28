package cmd

import (
	"context"
	"fmt"

	"github.com/ksysoev/omnidex/pkg/api"
	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/ksysoev/omnidex/pkg/prov/markdown"
	"github.com/ksysoev/omnidex/pkg/prov/openapi"
	"github.com/ksysoev/omnidex/pkg/repo/docstore"
	"github.com/ksysoev/omnidex/pkg/repo/search"
	"github.com/ksysoev/omnidex/pkg/views"
)

// RunCommand initializes the logger, loads configuration, creates the core and API services,
// and starts the API service. It returns an error if any step fails.
func RunCommand(ctx context.Context, flags *cmdFlags) error {
	if err := initLogger(flags); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	cfg, err := loadConfig(flags)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize document storage.
	store, err := docstore.New(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("failed to create document store: %w", err)
	}

	// Initialize search engine.
	searchEngine, err := search.NewBleve(cfg.Search.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to create search engine: %w", err)
	}

	defer searchEngine.Close()

	// Initialize markdown renderer.
	renderer := markdown.New()

	// Initialize OpenAPI processor.
	openapiProcessor := openapi.New()

	// Initialize core service with content processors.
	processors := map[core.ContentType]core.ContentProcessor{
		core.ContentTypeMarkdown: renderer,
		core.ContentTypeOpenAPI:  openapiProcessor,
	}

	svc := core.New(store, searchEngine, processors)

	// Initialize view renderer.
	viewRenderer := views.New()

	// Initialize and run API server.
	apiSvc, err := api.New(cfg.API, svc, viewRenderer)
	if err != nil {
		return fmt.Errorf("failed to create API service: %w", err)
	}

	err = apiSvc.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run API service: %w", err)
	}

	return nil
}
