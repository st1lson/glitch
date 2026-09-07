package engine

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/st1lson/glitch/internal/api"
	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/openapi"
	"github.com/st1lson/glitch/internal/proxy"
	"github.com/st1lson/glitch/internal/storage"
)

// Engine is the Strategy Interface for all backend modes.
type Engine interface {
	Name() string
	Resources() []string
	Handler() http.Handler
}

type OperationIndexer interface {
	OperationIndex() *chaos.OperationIndex
}

// New is the Factory Method that sniffs the configuration
// and returns the appropriate Engine strategy.
func New(targetFile, proxyURL string, readOnly bool) (Engine, error) {
	if proxyURL != "" {
		h, err := proxy.NewProxyHandler(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("proxy engine: %w", err)
		}
		return &ProxyEngine{targetURL: proxyURL, handler: h}, nil
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		return nil, fmt.Errorf("reading target file: %w", err)
	}

	if isOpenAPI(data) {
		mockRes, err := openapi.NewMock(targetFile)
		if err != nil {
			return nil, fmt.Errorf("openapi engine: %w", err)
		}
		return &OpenAPIEngine{file: targetFile, handler: mockRes.Handler, opIdx: mockRes.OperationIndex}, nil
	}

	store, err := storage.NewJSONStore(targetFile, readOnly)
	if err != nil {
		return nil, fmt.Errorf("json engine: %w", err)
	}
	r := chi.NewRouter()
	api.RegisterRoutes(r, store)

	return &JSONEngine{store: store, handler: r}, nil
}

func isOpenAPI(data []byte) bool {
	return bytes.Contains(data, []byte("openapi:")) ||
		bytes.Contains(data, []byte("\"openapi\":")) ||
		bytes.Contains(data, []byte("swagger:")) ||
		bytes.Contains(data, []byte("\"swagger\":"))
}

type ProxyEngine struct {
	targetURL string
	handler   http.Handler
}

func (e *ProxyEngine) Name() string          { return "Reverse Proxy" }
func (e *ProxyEngine) Resources() []string   { return []string{"Forwarding to " + e.targetURL} }
func (e *ProxyEngine) Handler() http.Handler { return e.handler }

type OpenAPIEngine struct {
	file    string
	handler http.Handler
	opIdx   *chaos.OperationIndex
}

func (e *OpenAPIEngine) Name() string                         { return "OpenAPI Mock Server" }
func (e *OpenAPIEngine) Resources() []string                  { return []string{"Mocking endpoints from " + e.file} }
func (e *OpenAPIEngine) Handler() http.Handler                { return e.handler }
func (e *OpenAPIEngine) OperationIndex() *chaos.OperationIndex { return e.opIdx }

type JSONEngine struct {
	store   storage.Store
	handler http.Handler
}

func (e *JSONEngine) Name() string          { return "JSON Database" }
func (e *JSONEngine) Resources() []string   { return e.store.Collections() }
func (e *JSONEngine) Handler() http.Handler { return e.handler }
