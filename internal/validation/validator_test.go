package validation_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
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

	t.Run("ValidateLegendStructure", func(t *testing.T) {
		if err := validator.ValidateLegendStructure(); err != nil {
			t.Errorf("Legend validation failed: %v", err)
		}
	})

	t.Run("ValidateExactRender", func(t *testing.T) {
		if err := validator.ValidateExactRender(); err != nil {
			t.Errorf("Exact render validation failed: %v", err)
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

	t.Run("ValidateEntrypointConnections", func(t *testing.T) {
		if err := validator.ValidateEntrypointConnections(); err != nil {
			t.Errorf("Entrypoint connection validation failed: %v", err)
		}
	})

	t.Run("ValidateServiceConnections", func(t *testing.T) {
		if err := validator.ValidateServiceConnections(); err != nil {
			t.Errorf("Service connection validation failed: %v", err)
		}
	})

	t.Run("ValidateEntrypointConnections", func(t *testing.T) {
		if err := validator.ValidateEntrypointConnections(); err != nil {
			t.Errorf("Entrypoint connection validation failed: %v", err)
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

	t.Run("ValidateLegendStructure", func(t *testing.T) {
		if err := validator.ValidateLegendStructure(); err != nil {
			t.Errorf("Legend validation failed: %v", err)
		}
	})

	t.Run("ValidateExactRender", func(t *testing.T) {
		if err := validator.ValidateExactRender(); err != nil {
			t.Errorf("Exact render validation failed: %v", err)
		}
	})

	t.Run("ValidateResources", func(t *testing.T) {
		if err := validator.ValidateResources(); err != nil {
			t.Errorf("Resource validation failed: %v", err)
		}
	})

	t.Run("ValidateEntrypointConnections", func(t *testing.T) {
		if err := validator.ValidateEntrypointConnections(); err != nil {
			t.Errorf("Entrypoint connection validation failed: %v", err)
		}
	})

	t.Run("ValidatePVCConnections", func(t *testing.T) {
		if err := validator.ValidatePVCConnections(); err != nil {
			t.Errorf("PVC connection validation failed: %v", err)
		}
	})
}

func TestD2Validator_EscapedIdentifiersAndLabels(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name:       "team.alpha",
			ConfigMaps: 1,
			Secrets:    2,
			Entrypoints: []model.Entrypoint{{
				Name:     "edge.v2",
				Kind:     "Ingress",
				Class:    "nginx.public",
				Hosts:    []string{"api.v2.example.com"},
				Services: []string{"api.v2-service"},
			}},
			Deployments: []model.Workload{{
				Name:     "api.v2",
				Kind:     "Deployment",
				Replicas: 2,
				Labels: map[string]string{
					"app.kubernetes.io/name": "api.v2",
				},
				VolumeMounts: []model.VolumeMount{{
					PVCName:   "cache.data",
					MountPath: "/var/lib/data",
				}},
			}},
			Services: []model.Service{{
				Name: "api.v2-service",
				Type: "ClusterIP",
				Selector: map[string]string{
					"app.kubernetes.io/name": "api.v2",
				},
			}},
			PVCs: []model.PVC{{
				Name: "cache.data",
			}},
		}},
	}

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}

	validator := validation.NewD2Validator(cluster, buf.String())

	if err := validator.ValidateSyntax(); err != nil {
		t.Fatalf("Syntax validation failed: %v", err)
	}
	if err := validator.ValidateLegendStructure(); err != nil {
		t.Fatalf("Legend validation failed: %v", err)
	}
	if err := validator.ValidateExactRender(); err != nil {
		t.Fatalf("Exact render validation failed: %v", err)
	}
	if err := validator.ValidateResources(); err != nil {
		t.Fatalf("Resource validation failed: %v", err)
	}
	if err := validator.ValidateWorkloadLabels(); err != nil {
		t.Fatalf("Workload label validation failed: %v", err)
	}
	if err := validator.ValidateEntrypointConnections(); err != nil {
		t.Fatalf("Entrypoint connection validation failed: %v", err)
	}
	if err := validator.ValidateServiceConnections(); err != nil {
		t.Fatalf("Service connection validation failed: %v", err)
	}
	if err := validator.ValidatePVCConnections(); err != nil {
		t.Fatalf("PVC connection validation failed: %v", err)
	}
	if err := validator.ValidateConfigInfo(); err != nil {
		t.Fatalf("Config info validation failed: %v", err)
	}
}

func TestD2Validator_ValidateExactRenderMatchesHandAuthoredOutput(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "apps",
			DaemonSets: []model.Workload{{
				Name:     "node-agent",
				Kind:     "DaemonSet",
				Replicas: 3,
				Labels: map[string]string{
					"app": "node-agent",
				},
			}},
			Services: []model.Service{{
				Name: "agent-svc",
				Type: "ClusterIP",
				Selector: map[string]string{
					"app": "node-agent",
				},
			}},
		}},
	}

	daemonSetID := render.WorkloadID(model.Workload{Name: "node-agent", Kind: "DaemonSet"})
	serviceID := render.ServiceID("agent-svc")
	actual := fmt.Sprintf(`# Generated by k8s-d2
direction: right

vars: {
  d2-legend: {
    daemonset: {
      label: "◈ DaemonSet"
      style.fill: "#f9f9f9"
    }

    service: {
      label: "⎈ Service"
      style.fill: "#cce5ff"
    }
  }
}

namespaces: {

  id_61707073: {
    label: "apps"
    style.fill: "#f0f0f0"

    %s: {
      label: "◈ node-agent"
    }
    %s: {
      label: "⎈ agent-svc\nClusterIP"
      style.fill: "#cce5ff"
    }
    %s -> %s
  }

}
`, daemonSetID, serviceID, serviceID, daemonSetID)

	validator := validation.NewD2Validator(cluster, actual)

	if err := validator.ValidateExactRender(); err != nil {
		t.Fatalf("expected hand-authored output to match exact renderer output: %v", err)
	}
}

func TestD2Validator_ValidateWorkloadLabelsRejectsDaemonSetReplicaDrift(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "apps",
			DaemonSets: []model.Workload{{
				Name:     "node-agent",
				Kind:     "DaemonSet",
				Replicas: 3,
			}},
		}},
	}

	actual := fmt.Sprintf("# Generated by k8s-d2\ndirection: right\n\nvars: {\n  d2-legend: {\n    daemonset: {\n      label: \"◈ DaemonSet\"\n      style.fill: \"#f9f9f9\"\n    }\n  }\n}\n\nnamespaces: {\n\n  id_61707073: {\n    label: \"apps\"\n    style.fill: \"#f0f0f0\"\n\n    %s: {\n      label: \"◈ node-agent (3)\"\n    }\n  }\n\n}\n", render.WorkloadID(model.Workload{Name: "node-agent", Kind: "DaemonSet"}))

	validator := validation.NewD2Validator(cluster, actual)

	if err := validator.ValidateWorkloadLabels(); err == nil {
		t.Fatalf("expected daemonset label validation to fail for replica suffix drift")
	}
	if err := validator.ValidateExactRender(); err == nil {
		t.Fatalf("expected exact render validation to fail for daemonset label drift")
	}
}

func TestD2Validator_ValidateExactRenderReportsFirstDifferingLine(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "apps",
			Deployments: []model.Workload{{
				Name:     "api",
				Kind:     "Deployment",
				Replicas: 1,
			}},
		}},
	}

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}

	actual := buf.String()
	actual = strings.Replace(actual, "label: \"● api (1)\"", "label: \"● api (2)\"", 1)

	validator := validation.NewD2Validator(cluster, actual)
	err := validator.ValidateExactRender()
	if err == nil {
		t.Fatalf("expected exact render validation to fail")
	}
	if !strings.Contains(err.Error(), "first difference at line") {
		t.Fatalf("expected exact render mismatch to report differing line, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected \"      label: \\\"● api (1)\\\"\"") {
		t.Fatalf("expected exact render mismatch to include expected line, got: %v", err)
	}
}

func TestD2Validator_ValidateResourcesScopesChecksToNamespaceBlock(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{
			{
				Name: "alpha",
				Deployments: []model.Workload{{
					Name:     "shared-api",
					Kind:     "Deployment",
					Replicas: 1,
				}},
			},
			{
				Name: "beta",
				Deployments: []model.Workload{{
					Name:     "shared-api",
					Kind:     "Deployment",
					Replicas: 1,
				}},
				ConfigMaps: 1,
			},
		},
	}

	actual := fmt.Sprintf(`# Generated by k8s-d2
direction: right

vars: {
  d2-legend: {
    deployment: {
      label: "● Deployment"
      style.fill: "#f9f9f9"
    }

    config: {
      label: "ConfigMaps | Secrets"
      style.fill: "#ffffcc"
    }
  }
}

namespaces: {

  id_616c706861: {
    label: "alpha"
    style.fill: "#f0f0f0"
  }

  id_62657461: {
    label: "beta"
    style.fill: "#f0f0f0"

    %s: {
      label: "● shared-api (1)"
    }
    _config: {
      label: "CM: 1 | Sec: 0"
      style.fill: "#ffffcc"
    }
  }

}
`, render.WorkloadID(model.Workload{Name: "shared-api", Kind: "Deployment"}))

	validator := validation.NewD2Validator(cluster, actual)

	err := validator.ValidateResources()
	if err == nil {
		t.Fatalf("expected namespace-scoped resource validation to fail")
	}
	if !strings.Contains(err.Error(), "missing workload: shared-api") {
		t.Fatalf("expected missing workload error for alpha namespace, got: %v", err)
	}
}

func TestD2Validator_ValidateResourcesDistinguishesWorkloadKindsWithSameName(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "apps",
			Deployments: []model.Workload{{
				Name:     "api",
				Kind:     "Deployment",
				Replicas: 2,
			}},
			StatefulSets: []model.Workload{{
				Name:     "api",
				Kind:     "StatefulSet",
				Replicas: 1,
			}},
		}},
	}

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}

	actual := strings.Replace(
		buf.String(),
		fmt.Sprintf(
			"    %s: {\n      label: %s\n    }\n",
			render.WorkloadID(model.Workload{Name: "api", Kind: "StatefulSet"}),
			render.QuoteString(render.WorkloadLabel(cluster.Namespaces[0].StatefulSets[0])),
		),
		"",
		1,
	)

	validator := validation.NewD2Validator(cluster, actual)
	err := validator.ValidateResources()
	if err == nil {
		t.Fatalf("expected resource validation to fail when one same-name workload kind is missing")
	}
	if !strings.Contains(err.Error(), "missing workload: api (StatefulSet)") {
		t.Fatalf("expected missing statefulset workload error, got: %v", err)
	}
}

func TestD2Validator_ValidateConfigInfoScopesChecksToNamespaceBlock(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{
			{
				Name:       "alpha",
				ConfigMaps: 1,
			},
			{
				Name:       "beta",
				ConfigMaps: 1,
				Secrets:    2,
			},
		},
	}

	actual := `# Generated by k8s-d2
direction: right

vars: {
  d2-legend: {
    config: {
      label: "ConfigMaps | Secrets"
      style.fill: "#ffffcc"
    }
  }
}

namespaces: {

  id_616c706861: {
    label: "alpha"
    style.fill: "#f0f0f0"
  }

  id_62657461: {
    label: "beta"
    style.fill: "#f0f0f0"

    _config: {
      label: "CM: 1 | Sec: 2"
      style.fill: "#ffffcc"
    }
  }

}
`

	validator := validation.NewD2Validator(cluster, actual)

	err := validator.ValidateConfigInfo()
	if err == nil {
		t.Fatalf("expected namespace-scoped config validation to fail")
	}
	if !strings.Contains(err.Error(), "incorrect config info for namespace alpha") {
		t.Fatalf("expected config info error for alpha namespace, got: %v", err)
	}
}

func TestD2Validator_ValidateEntrypointConnectionsScopesChecksToNamespaceBlock(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{
			{
				Name: "alpha",
				Entrypoints: []model.Entrypoint{{
					Name:     "public-edge",
					Kind:     "Ingress",
					Services: []string{"web-service"},
				}},
				Services: []model.Service{{
					Name: "web-service",
					Type: "ClusterIP",
				}},
			},
			{
				Name: "beta",
				Entrypoints: []model.Entrypoint{{
					Name:     "public-edge",
					Kind:     "Ingress",
					Services: []string{"web-service"},
				}},
				Services: []model.Service{{
					Name: "web-service",
					Type: "ClusterIP",
				}},
			},
		},
	}

	actual := `# Generated by k8s-d2
direction: right

vars: {
  d2-legend: {
    entrypoint: {
      label: "⇢ Entrypoint"
      style.fill: "#ffe6cc"
    }

    service: {
      label: "⎈ Service"
      style.fill: "#cce5ff"
    }
  }
}

namespaces: {

  id_616c706861: {
    label: "alpha"
    style.fill: "#f0f0f0"
  }

  id_62657461: {
    label: "beta"
    style.fill: "#f0f0f0"

    ep_id_696e67726573733a7075626c69632d65646765: {
      label: "⇢ public-edge\nIngress"
      style.fill: "#ffe6cc"
    }
    svc_id_7765622d73657276696365: {
      label: "⎈ web-service\nClusterIP"
      style.fill: "#cce5ff"
    }
    ep_id_696e67726573733a7075626c69632d65646765 -> svc_id_7765622d73657276696365
  }

}
`

	validator := validation.NewD2Validator(cluster, actual)

	err := validator.ValidateEntrypointConnections()
	if err == nil {
		t.Fatalf("expected namespace-scoped entrypoint validation to fail")
	}
	if !strings.Contains(err.Error(), "missing expected connection") {
		t.Fatalf("expected missing entrypoint connection error for alpha namespace, got: %v", err)
	}
}

func TestParseTestFixtures_ExcludesStatefulSetGeneratedPVCsWithoutStorage(t *testing.T) {
	data, err := loadBaseFixtures()
	if err != nil {
		t.Fatalf("Failed to load base fixtures: %v", err)
	}

	cluster, err := parseTestFixtures(data, false)
	if err != nil {
		t.Fatalf("Failed to parse base fixtures: %v", err)
	}

	if len(cluster.Namespaces) != 1 {
		t.Fatalf("expected one namespace, got %d", len(cluster.Namespaces))
	}

	for _, pvc := range cluster.Namespaces[0].PVCs {
		if strings.HasPrefix(pvc.Name, "data-database-") {
			t.Fatalf("did not expect generated statefulset PVC %q without storage parsing", pvc.Name)
		}
	}
}

func TestParseTestFixtures_IncludesStatefulSetGeneratedPVCsWithStorage(t *testing.T) {
	data, err := loadAllFixtures()
	if err != nil {
		t.Fatalf("Failed to load all fixtures: %v", err)
	}

	cluster, err := parseTestFixtures(data, true)
	if err != nil {
		t.Fatalf("Failed to parse all fixtures: %v", err)
	}

	if len(cluster.Namespaces) != 1 {
		t.Fatalf("expected one namespace, got %d", len(cluster.Namespaces))
	}

	pvcNames := make(map[string]struct{}, len(cluster.Namespaces[0].PVCs))
	for _, pvc := range cluster.Namespaces[0].PVCs {
		pvcNames[pvc.Name] = struct{}{}
	}

	for _, name := range []string{"logs-volume", "data-database-0", "data-database-1"} {
		if _, ok := pvcNames[name]; !ok {
			t.Fatalf("expected PVC %q in parsed storage fixtures", name)
		}
	}
}

func TestParseTestFixtures_WithStorage_RendersPVCsWithoutStorageClassNodes(t *testing.T) {
	data, err := loadAllFixtures()
	if err != nil {
		t.Fatalf("Failed to load all fixtures: %v", err)
	}

	cluster, err := parseTestFixtures(data, true)
	if err != nil {
		t.Fatalf("Failed to parse all fixtures: %v", err)
	}

	if len(cluster.Namespaces) != 1 {
		t.Fatalf("expected one namespace, got %d", len(cluster.Namespaces))
	}

	expectedStorageClassLabels := 0
	for _, pvc := range cluster.Namespaces[0].PVCs {
		if pvc.StorageClass == "standard" {
			expectedStorageClassLabels++
		}
	}
	if expectedStorageClassLabels == 0 {
		t.Fatalf("expected parsed storage fixtures to preserve PVC storage class metadata")
	}

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("Failed to render D2: %v", err)
	}

	output := buf.String()
	if got := strings.Count(output, "[standard]"); got != expectedStorageClassLabels {
		t.Fatalf("expected %d PVC storage class labels in output, got %d\n%s", expectedStorageClassLabels, got, output)
	}
	if strings.Contains(output, render.SanitizeID("standard")) {
		t.Fatalf("did not expect storage class %q to render as a standalone node\n%s", "standard", output)
	}
	if strings.Contains(output, "label: \"standard\"") {
		t.Fatalf("did not expect storage class %q to render as a standalone label\n%s", "standard", output)
	}
	if strings.Contains(output, "StorageClass") {
		t.Fatalf("did not expect StorageClass resources to render as nodes\n%s", output)
	}
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
		"07-ingress.yaml",
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
func parseTestFixtures(fixtureData [][]byte, includeStorage bool) (*model.Cluster, error) {
	parser := validation.NewFixtureParser(testNamespace, includeStorage)
	return parser.ParseFixtures(fixtureData)
}

// loadAndParseBaseFixtures is a convenience function that loads and parses base fixtures
func loadAndParseBaseFixtures() (*model.Cluster, error) {
	data, err := loadBaseFixtures()
	if err != nil {
		return nil, err
	}
	return parseTestFixtures(data, false)
}

// loadAndParseAllFixtures is a convenience function that loads and parses all fixtures
func loadAndParseAllFixtures() (*model.Cluster, error) {
	data, err := loadAllFixtures()
	if err != nil {
		return nil, err
	}
	return parseTestFixtures(data, true)
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
