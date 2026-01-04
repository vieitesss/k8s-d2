package validation_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
	"github.com/vieitesss/k8s-d2/pkg/validation"
)

func TestD2Validator_BasicTopology(t *testing.T) {
	// Load and parse base fixtures
	expectedCluster, err := loadAndParseBaseFixtures()
	if err != nil {
		t.Fatalf("Failed to load and parse fixtures: %v", err)
	}

	// Render D2 output
	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(expectedCluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}
	d2Output := buf.String()

	// Validate D2 output
	validator := validation.NewD2Validator(expectedCluster, d2Output)

	t.Run("ValidateSyntax", func(t *testing.T) {
		if err := validator.ValidateSyntax(); err != nil {
			t.Errorf("Syntax validation failed: %v", err)
		}
	})

	t.Run("ValidateResources", func(t *testing.T) {
		if err := validator.ValidateResources(); err != nil {
			t.Errorf("Resource validation failed: %v", err)
		}
	})

	t.Run("ValidateWorkloadLabels", func(t *testing.T) {
		if err := validator.ValidateWorkloadLabels(); err != nil {
			t.Errorf("Workload label validation failed: %v", err)
		}
	})

	t.Run("ValidateServiceConnections", func(t *testing.T) {
		if err := validator.ValidateServiceConnections(); err != nil {
			t.Errorf("Service connection validation failed: %v", err)
		}
	})

	t.Run("ValidateConfigInfo", func(t *testing.T) {
		if err := validator.ValidateConfigInfo(); err != nil {
			t.Errorf("Config info validation failed: %v", err)
		}
	})
}

func TestD2Validator_WithStorage(t *testing.T) {
	// Load and parse all fixtures including storage
	expectedCluster, err := loadAndParseAllFixtures()
	if err != nil {
		t.Fatalf("Failed to load and parse fixtures: %v", err)
	}

	// Render D2 output
	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(expectedCluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}
	d2Output := buf.String()

	// Validate D2 output
	validator := validation.NewD2Validator(expectedCluster, d2Output)

	t.Run("ValidateSyntax", func(t *testing.T) {
		if err := validator.ValidateSyntax(); err != nil {
			t.Errorf("Syntax validation failed: %v", err)
		}
	})

	t.Run("ValidateResources", func(t *testing.T) {
		if err := validator.ValidateResources(); err != nil {
			t.Errorf("Resource validation failed: %v", err)
		}
	})

	t.Run("ValidatePVCConnections", func(t *testing.T) {
		if err := validator.ValidatePVCConnections(); err != nil {
			t.Errorf("PVC connection validation failed: %v", err)
		}
	})
}

// Test constants
const (
	testNamespace = "k8s-d2-test"
)

// Fixture lists
var (
	baseFixtures = []string{
		"01-namespace.yaml",
		"02-configmaps-secrets.yaml",
		"03-deployments.yaml",
		"04-statefulsets.yaml",
		"05-daemonsets.yaml",
		"06-services.yaml",
	}

	storageFixtures = []string{
		"01-storageclass.yaml",
		"02-pvcs.yaml",
	}

	allFixtures = map[string][]string{
		"base":    baseFixtures,
		"storage": storageFixtures,
	}
)

// loadBaseFixtures loads only the base fixtures
func loadBaseFixtures() ([][]byte, error) {
	return loadFixtures("base", baseFixtures)
}

// loadAllFixtures loads both base and storage fixtures
func loadAllFixtures() ([][]byte, error) {
	var fixtureData [][]byte
	for dir, files := range allFixtures {
		data, err := loadFixtures(dir, files)
		if err != nil {
			return nil, err
		}
		fixtureData = append(fixtureData, data...)
	}
	return fixtureData, nil
}

// parseTestFixtures parses fixture data into a cluster model
func parseTestFixtures(fixtureData [][]byte) (*model.Cluster, error) {
	parser := validation.NewFixtureParser(testNamespace)
	return parser.ParseFixtures(fixtureData)
}

// loadAndParseBaseFixtures is a convenience function that loads and parses base fixtures
func loadAndParseBaseFixtures() (*model.Cluster, error) {
	data, err := loadBaseFixtures()
	if err != nil {
		return nil, err
	}
	return parseTestFixtures(data)
}

// loadAndParseAllFixtures is a convenience function that loads and parses all fixtures
func loadAndParseAllFixtures() (*model.Cluster, error) {
	data, err := loadAllFixtures()
	if err != nil {
		return nil, err
	}
	return parseTestFixtures(data)
}

// loadFixtures reads fixture files from test/fixtures/<dir>/
func loadFixtures(dir string, files []string) ([][]byte, error) {
	var data [][]byte

	// Get the project root directory (assuming we're in pkg/validation/)
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		path := filepath.Join(projectRoot, "test", "fixtures", dir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		data = append(data, content)
	}

	return data, nil
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
