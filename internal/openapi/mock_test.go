package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewMockHandler_Success(t *testing.T) {
	yamlSpec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: Get pets
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
                    name:
                      type: string
                      example: "Fido"
`

	jsonSpec := `
{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {
    "/pets": {
      "get": {
        "summary": "Get pets",
        "responses": {
          "200": {
            "description": "A list of pets",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "id": { "type": "integer" },
                      "name": { "type": "string", "example": "Fido" }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
`

	tests := []struct {
		name      string
		extension string
		content   string
	}{
		{"YAML Spec", "*.yaml", yamlSpec},
		{"JSON Spec", "*.json", jsonSpec},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "openapi-test-"+tt.extension)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			handler, err := NewMockHandler(tmpfile.Name())
			if err != nil {
				t.Fatalf("Failed to create mock handler: %v", err)
			}

			req := httptest.NewRequest("GET", "/pets", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
			}

			var pets []map[string]any
			if err := json.NewDecoder(rr.Body).Decode(&pets); err != nil {
				t.Fatalf("Failed to decode JSON response: %v", err)
			}

			if len(pets) == 0 {
				t.Errorf("Expected at least one mock pet in response array")
			} else {
				// Just ensure 'name' field is present in the mocked object
				if _, ok := pets[0]["name"]; !ok {
					t.Errorf("Expected mocked object to have 'name' field")
				}
			}
		})
	}
}

func TestNewMockHandler_InvalidFile(t *testing.T) {
	_, err := NewMockHandler("/tmp/does-not-exist.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
