package cmd

import (
	"context"
	"fmt"

	omnidex "github.com/ksysoev/omnidex"
	"github.com/ksysoev/omnidex/pkg/api"
	"github.com/ksysoev/omnidex/pkg/core"
	"github.com/ksysoev/omnidex/pkg/prov/markdown"
	"github.com/ksysoev/omnidex/pkg/prov/openapi"
	"github.com/ksysoev/omnidex/pkg/repo/docstore"
	"github.com/ksysoev/omnidex/pkg/repo/s3store"
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

	// Initialize search engine based on configured backend.
	var searchEng interface {
		Index(ctx context.Context, doc core.Document, plainText string) error
		Remove(ctx context.Context, docID string) error
		Search(ctx context.Context, query string, opts core.SearchOpts) (*core.SearchResults, error)
		ListByRepo(ctx context.Context, repo string) ([]string, error)
	}

	switch cfg.Search.Type {
	case "elasticsearch":
		searchEng, err = search.NewElastic(&cfg.Search.Elastic)
		if err != nil {
			return fmt.Errorf("failed to create elasticsearch engine: %w", err)
		}
	case "", "bleve":
		bleveEng, bleveErr := search.NewBleve(cfg.Search.IndexPath)
		if bleveErr != nil {
			return fmt.Errorf("failed to create search engine: %w", bleveErr)
		}

		defer bleveEng.Close()

		searchEng = bleveEng
	default:
		return fmt.Errorf("unknown search type %q: must be \"bleve\" or \"elasticsearch\"", cfg.Search.Type)
	}

	searchEngine := searchEng

	// Initialize markdown renderer.
	renderer := markdown.New()

	// Initialize OpenAPI processor.
	openapiProcessor := openapi.New()

	// Initialize core service with content processors.
	processors := map[core.ContentType]core.ContentProcessor{
		core.ContentTypeMarkdown: renderer,
		core.ContentTypeOpenAPI:  openapiProcessor,
	}

	// Initialize document storage backend selected by configuration and wire the core service.
	var svc *core.Service

	switch cfg.Storage.Type {
	case "s3":
		s3Store, err := s3store.New(ctx, cfg.Storage.S3)
		if err != nil {
			return fmt.Errorf("failed to create S3 document store: %w", err)
		}

		svc = core.New(s3Store, searchEngine, processors)
	case "", "local":
		localStore, err := docstore.New(cfg.Storage.Path)
		if err != nil {
			return fmt.Errorf("failed to create document store: %w", err)
		}

		svc = core.New(localStore, searchEngine, processors)
	default:
		return fmt.Errorf("unknown storage type %q: must be \"local\" or \"s3\"", cfg.Storage.Type)
	}

	// Initialize view renderer.
	viewRenderer := views.New()

	// Initialize and run API server.
	cfg.API.StaticFS = omnidex.StaticFiles

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
