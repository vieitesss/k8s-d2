package render

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/vieitesss/k8s-d2/pkg/model"
)

func TestRenderLegend_IncludesOnlyRenderedResourceTypes(t *testing.T) {
	tests := []struct {
		name       string
		cluster    *model.Cluster
		expected   []string
		unexpected []string
		wantLegend bool
	}{
		{
			name: "deployment only",
			cluster: &model.Cluster{
				Namespaces: []model.Namespace{{
					Name: "apps",
					Deployments: []model.Workload{{
						Name:     "web",
						Kind:     "Deployment",
						Replicas: 1,
					}},
				}},
			},
			expected:   []string{"deployment"},
			unexpected: []string{"statefulset", "daemonset", "service", "config", "pvc"},
			wantLegend: true,
		},
		{
			name: "service and config only",
			cluster: &model.Cluster{
				Namespaces: []model.Namespace{{
					Name: "apps",
					Services: []model.Service{{
						Name: "web-service",
						Type: "ClusterIP",
					}},
					ConfigMaps: 1,
				}},
			},
			expected:   []string{"service", "config"},
			unexpected: []string{"deployment", "statefulset", "daemonset", "pvc"},
			wantLegend: true,
		},
		{
			name: "pvc only",
			cluster: &model.Cluster{
				Namespaces: []model.Namespace{{
					Name: "storage",
					PVCs: []model.PVC{{
						Name: "data",
					}},
				}},
			},
			expected:   []string{"pvc"},
			unexpected: []string{"deployment", "statefulset", "daemonset", "service", "config"},
			wantLegend: true,
		},
		{
			name: "empty namespace omits legend",
			cluster: &model.Cluster{
				Namespaces: []model.Namespace{{
					Name: "empty",
				}},
			},
			unexpected: []string{"deployment", "statefulset", "daemonset", "service", "config", "pvc"},
			wantLegend: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderTestCluster(t, tt.cluster)

			if !tt.wantLegend {
				if strings.Contains(output, "d2-legend:") {
					t.Fatalf("expected legend to be omitted, output was:\n%s", output)
				}
				return
			}

			if !strings.Contains(output, "d2-legend:") {
				t.Fatalf("expected built-in legend block, output was:\n%s", output)
			}

			varsBlock := extractBlockFromMarker(t, output, "vars:")
			if !strings.Contains(varsBlock, "d2-legend:") {
				t.Fatalf("expected legend inside vars block, output was:\n%s", output)
			}

			legendBlock := extractBlockFromMarker(t, output, "d2-legend:")
			for _, entry := range tt.expected {
				if !strings.Contains(legendBlock, fmt.Sprintf("%s: {", entry)) {
					t.Errorf("expected legend entry %q in block:\n%s", entry, legendBlock)
				}
			}
			for _, entry := range tt.unexpected {
				if strings.Contains(legendBlock, fmt.Sprintf("%s: {", entry)) {
					t.Errorf("did not expect legend entry %q in block:\n%s", entry, legendBlock)
				}
			}
		})
	}
}

func TestRender_IgnoresDeprecatedGridColumns(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{
			{
				Name: "vars",
				Deployments: []model.Workload{{
					Name:     "web",
					Kind:     "Deployment",
					Replicas: 1,
				}},
			},
			{
				Name: "infra",
				Services: []model.Service{{
					Name: "metrics",
					Type: "ClusterIP",
				}},
			},
		},
	}

	var buf bytes.Buffer
	renderer := NewD2Renderer(&buf, 4)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "grid-columns:") {
		t.Fatalf("expected automatic layout without grid constraints, output was:\n%s", output)
	}
	if !strings.Contains(output, "namespaces: {") {
		t.Fatalf("expected namespaces container in output, output was:\n%s", output)
	}
	if !strings.Contains(output, "  vars: {") {
		t.Fatalf("expected namespace to render inside namespaces container, output was:\n%s", output)
	}
}

func renderTestCluster(t *testing.T, cluster *model.Cluster) string {
	t.Helper()

	var buf bytes.Buffer
	renderer := NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	return buf.String()
}

func extractBlockFromMarker(t *testing.T, input, marker string) string {
	t.Helper()

	markerIndex := strings.Index(input, marker)
	if markerIndex == -1 {
		t.Fatalf("missing marker %q in output:\n%s", marker, input)
	}

	blockStart := strings.Index(input[markerIndex:], "{")
	if blockStart == -1 {
		t.Fatalf("missing opening brace for marker %q in output:\n%s", marker, input)
	}

	braceDepth := 0
	for i, r := range input[markerIndex+blockStart:] {
		switch r {
		case '{':
			braceDepth++
		case '}':
			braceDepth--
			if braceDepth == 0 {
				return input[markerIndex : markerIndex+blockStart+i+1]
			}
		}
	}

	t.Fatalf("unclosed block for marker %q in output:\n%s", marker, input)
	return ""
}
