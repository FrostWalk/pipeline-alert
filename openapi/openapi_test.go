package openapi

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSpecLoadsAndValidates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	specPath := filepath.Join(filepath.Dir(thisFile), "openapi.yaml")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(raw)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("validate spec: %v", err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %q", doc.OpenAPI)
	}
}
