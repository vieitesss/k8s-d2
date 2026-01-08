package validation_test

import (
	"os"
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
)

// TestD2Output_BasicFromEnv validates D2 output provided via environment variables
// This is used by Dagger to validate actual k8sdd output against real cluster
func TestD2Output_BasicFromEnv(t *testing.T) {
	d2Output := os.Getenv("D2_OUTPUT_BASIC")
	if d2Output == "" {
		t.Skip("D2_OUTPUT_BASIC not set, skipping integration test")
	}

	// Load and parse base fixtures
	expectedCluster, err := loadAndParseBaseFixtures()
	if err != nil {
		t.Fatalf("Failed to load and parse fixtures: %v", err)
	}

	// Validate D2 output
	validator := validation.NewD2Validator(expectedCluster, d2Output)

	if err := validator.ValidateSyntax(); err != nil {
		t.Errorf("Syntax validation failed: %v", err)
	}

	if err := validator.ValidateResources(); err != nil {
		t.Errorf("Resource validation failed: %v", err)
	}

	if err := validator.ValidateWorkloadLabels(); err != nil {
		t.Errorf("Workload label validation failed: %v", err)
	}

	if err := validator.ValidateServiceConnections(); err != nil {
		t.Errorf("Service connection validation failed: %v", err)
	}

	if err := validator.ValidateConfigInfo(); err != nil {
		t.Errorf("Config info validation failed: %v", err)
	}
}

// TestD2Output_StorageFromEnv validates D2 output with storage layer
func TestD2Output_StorageFromEnv(t *testing.T) {
	d2Output := os.Getenv("D2_OUTPUT_STORAGE")
	if d2Output == "" {
		t.Skip("D2_OUTPUT_STORAGE not set, skipping integration test")
	}

	// Load and parse all fixtures including storage
	expectedCluster, err := loadAndParseAllFixtures()
	if err != nil {
		t.Fatalf("Failed to load and parse fixtures: %v", err)
	}

	// Validate D2 output
	validator := validation.NewD2Validator(expectedCluster, d2Output)

	if err := validator.ValidateSyntax(); err != nil {
		t.Errorf("Syntax validation failed: %v", err)
	}

	if err := validator.ValidateResources(); err != nil {
		t.Errorf("Resource validation failed: %v", err)
	}

	if err := validator.ValidatePVCConnections(); err != nil {
		t.Errorf("PVC connection validation failed: %v", err)
	}
}

// TestD2Output_QuietMode validates that quiet mode produces identical output
func TestD2Output_QuietMode(t *testing.T) {
	basicOutput := os.Getenv("D2_OUTPUT_BASIC")
	quietOutput := os.Getenv("D2_OUTPUT_QUIET")

	if basicOutput == "" || quietOutput == "" {
		t.Skip("D2_OUTPUT_BASIC or D2_OUTPUT_QUIET not set, skipping quiet mode test")
	}

	if basicOutput != quietOutput {
		t.Errorf("Quiet mode output differs from normal mode")
		t.Logf("Basic output length: %d", len(basicOutput))
		t.Logf("Quiet output length: %d", len(quietOutput))
	}
}
