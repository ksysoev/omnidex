//go:build !compile

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ksysoev/omnidex/pkg/repo/docstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAssetPage_Success(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{
		svc:    svc,
		views:  views,
		config: Config{APIKeys: []string{"test-key"}},
	}

	mux := api.newMux()

	imgData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	svc.EXPECT().GetAsset(mock.Anything, "owner/repo", "images/arch.png").Return(imgData, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/owner/repo/images/arch.png", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
	assert.Equal(t, imgData, rec.Body.Bytes())
}

func TestAssetPage_NotFound(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{
		svc:    svc,
		views:  views,
		config: Config{APIKeys: []string{"test-key"}},
	}

	mux := api.newMux()

	svc.EXPECT().GetAsset(mock.Anything, "owner/repo", "missing.png").Return(nil, docstore.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/assets/owner/repo/missing.png", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssetPage_InternalError(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{
		svc:    svc,
		views:  views,
		config: Config{APIKeys: []string{"test-key"}},
	}

	mux := api.newMux()

	svc.EXPECT().GetAsset(mock.Anything, "owner/repo", "broken.png").Return(nil, errors.New("disk failure"))

	req := httptest.NewRequest(http.MethodGet, "/assets/owner/repo/broken.png", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAssetPage_SVGContentType(t *testing.T) {
	svc := NewMockService(t)
	views := NewMockViewRenderer(t)

	api := &API{
		svc:    svc,
		views:  views,
		config: Config{APIKeys: []string{"test-key"}},
	}

	mux := api.newMux()

	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	svc.EXPECT().GetAsset(mock.Anything, "owner/repo", "diagram.svg").Return(svgData, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/owner/repo/diagram.svg", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/svg+xml", rec.Header().Get("Content-Type"))
}
