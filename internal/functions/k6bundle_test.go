package functions_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/functions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestK6BundleFunction_Basic(t *testing.T) {
	tempDir := t.TempDir()

	libFile := filepath.Join(tempDir, "utils.js")
	if err := os.WriteFile(libFile, []byte(`export const sum = (a, b) => a + b;`), 0644); err != nil {
		t.Fatalf("Failed to create lib file: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.js")
	testContent := `
import { check } from 'k6';
import { sum } from './utils.js';

export default function() {
    const result = sum(1, 2);
    check(result, { 'sum works': (r) => r === 3 });
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	f := functions.NewK6BundleFunction()
	args := function.NewArgumentsData([]attr.Value{types.StringValue(testFile)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringUnknown())}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	bundledCode := resp.Result.Value().(types.String).ValueString()
	if !strings.Contains(bundledCode, "sum") || !strings.Contains(bundledCode, "check") {
		t.Error("Bundled code should contain both sum function and k6 check")
	}
}

func TestK6BundleFunction_FileNotFound(t *testing.T) {
	f := functions.NewK6BundleFunction()
	args := function.NewArgumentsData([]attr.Value{types.StringValue("/nonexistent.js")})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringUnknown())}

	f.Run(context.Background(), req, resp)

	if resp.Error == nil {
		t.Fatal("Expected error for nonexistent file")
	}
	if !strings.Contains(resp.Error.Error(), "File does not exist") {
		t.Errorf("Expected 'File does not exist' error, got: %s", resp.Error.Error())
	}
}
