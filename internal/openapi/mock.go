package openapi

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/renderer"
)

// NewMockHandler loads an OpenAPI spec from filePath and returns an http.Handler 
// that mounts all defined paths and generates mock responses on the fly.
func NewMockHandler(filePath string) (http.Handler, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	doc, err := libopenapi.NewDocument(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create openapi document: %w", err)
	}

	v3Model, errs := doc.BuildV3Model()
	if v3Model == nil {
		return nil, fmt.Errorf("failed to build v3 model: %v", errs)
	}

	r := chi.NewRouter()

	if v3Model.Model.Paths == nil || v3Model.Model.Paths.PathItems == nil {
		return r, nil
	}

	for pair := v3Model.Model.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
		path := pair.Key()
		pathItem := pair.Value()

		ops := map[string]*v3.Operation{
			http.MethodGet:    pathItem.Get,
			http.MethodPost:   pathItem.Post,
			http.MethodPut:    pathItem.Put,
			http.MethodDelete: pathItem.Delete,
			http.MethodPatch:  pathItem.Patch,
		}

		for method, operation := range ops {
			if operation == nil {
				continue
			}

			m := method
			p := path
			op := operation

			r.Method(m, p, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				handleMockRequest(w, req, op)
			}))
		}
	}

	return r, nil
}

func handleMockRequest(w http.ResponseWriter, r *http.Request, op *v3.Operation) {
	if op.Responses == nil {
		writeNoSchema(w)
		return
	}

	var foundResp *v3.Response
	var statusCode = 200

	if op.Responses.Codes != nil {
		for codePair := op.Responses.Codes.First(); codePair != nil; codePair = codePair.Next() {
			codeStr := codePair.Key()
			resp := codePair.Value()
			if strings.HasPrefix(codeStr, "2") {
				foundResp = resp
				fmt.Sscanf(codeStr, "%d", &statusCode)
				break
			}
		}
	}

	if foundResp == nil && op.Responses.Default != nil {
		foundResp = op.Responses.Default
	}

	if foundResp == nil {
		writeNoSchema(w)
		return
	}

	if statusCode == 204 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if foundResp.Content == nil {
		writeNoSchema(w)
		return
	}

	var jsonMediaType *v3.MediaType
	for mtPair := foundResp.Content.First(); mtPair != nil; mtPair = mtPair.Next() {
		if strings.Contains(mtPair.Key(), "json") {
			jsonMediaType = mtPair.Value()
			break
		}
	}

	if jsonMediaType == nil || jsonMediaType.Schema == nil || jsonMediaType.Schema.Schema() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"mock": "No schema defined"}`))
		return
	}

	mg := renderer.NewMockGenerator(renderer.JSON)
	mockBytes, err := mg.GenerateMock(jsonMediaType.Schema.Schema(), "")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error": "Failed to generate mock: %v"}`, err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(mockBytes)
}

func writeNoSchema(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"mock": "No schema defined"}`))
}
