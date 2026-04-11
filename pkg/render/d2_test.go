package render

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vieitesss/k8s-d2/pkg/model"
)

func TestFormatServicePortLabel(t *testing.T) {
	tests := []struct {
		name string
		port model.Port
		want string
	}{
		{name: "same target port omits arrow", port: model.Port{Port: 80, TargetPort: "80"}, want: "80"},
		{name: "named target port kept", port: model.Port{Name: "http", Port: 80, TargetPort: "web"}, want: "http: 80 -> web"},
		{name: "numeric target port kept", port: model.Port{Name: "metrics", Port: 9090, TargetPort: "9091"}, want: "metrics: 9090 -> 9091"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.FormatServicePortLabel(tt.port); got != tt.want {
				t.Fatalf("FormatServicePortLabel(%+v) = %q, want %q", tt.port, got, tt.want)
			}
		})
	}
}

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
	if !strings.Contains(output, fmt.Sprintf("  %s: {", SanitizeID("vars"))) {
		t.Fatalf("expected namespace to render inside namespaces container, output was:\n%s", output)
	}
}

func TestRender_SortsNamespacesAndResourcesDeterministically(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{
			{
				Name: "zeta",
				Deployments: []model.Workload{
					{Name: "web", Kind: "Deployment", Replicas: 2},
				},
			},
			{
				Name: "alpha",
				Deployments: []model.Workload{
					{Name: "z-api", Kind: "Deployment", Replicas: 2},
					{Name: "a-api", Kind: "Deployment", Replicas: 1},
				},
				StatefulSets: []model.Workload{
					{Name: "z-db", Kind: "StatefulSet", Replicas: 1},
					{Name: "a-db", Kind: "StatefulSet", Replicas: 1},
				},
				DaemonSets: []model.Workload{
					{Name: "z-agent", Kind: "DaemonSet", Replicas: 3},
					{Name: "a-agent", Kind: "DaemonSet", Replicas: 3},
				},
				Services: []model.Service{
					{Name: "z-service", Type: "ClusterIP"},
					{Name: "a-service", Type: "ClusterIP"},
				},
				PVCs: []model.PVC{
					{Name: "z-data"},
					{Name: "a-data"},
				},
			},
		},
	}

	output := renderTestCluster(t, cluster)

	assertAppearsInOrder(t, output,
		fmt.Sprintf("  %s: {", SanitizeID("alpha")),
		fmt.Sprintf("  %s: {", SanitizeID("zeta")),
	)

	alphaBlock := extractBlockFromMarker(t, output, fmt.Sprintf("  %s:", SanitizeID("alpha")))
	assertAppearsInOrder(t, alphaBlock,
		fmt.Sprintf("  %s: {", SanitizeID("a-api")),
		fmt.Sprintf("  %s: {", SanitizeID("z-api")),
	)
	assertAppearsInOrder(t, alphaBlock,
		fmt.Sprintf("  %s: {", SanitizeID("a-db")),
		fmt.Sprintf("  %s: {", SanitizeID("z-db")),
	)
	assertAppearsInOrder(t, alphaBlock,
		fmt.Sprintf("  %s: {", SanitizeID("a-agent")),
		fmt.Sprintf("  %s: {", SanitizeID("z-agent")),
	)
	assertAppearsInOrder(t, alphaBlock,
		fmt.Sprintf("  %s: {", ServiceID("a-service")),
		fmt.Sprintf("  %s: {", ServiceID("z-service")),
	)
	assertAppearsInOrder(t, alphaBlock,
		fmt.Sprintf("  %s: {", PVCID("a-data")),
		fmt.Sprintf("  %s: {", PVCID("z-data")),
	)
}

func TestRender_SortsWorkloadPVCConnectionsDeterministically(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "apps",
			Deployments: []model.Workload{{
				Name:     "web",
				Kind:     "Deployment",
				Replicas: 1,
				VolumeMounts: []model.VolumeMount{
					{PVCName: "z-cache", MountPath: "/cache"},
					{PVCName: "a-data", MountPath: "/data", ReadOnly: true},
				},
			}},
			PVCs: []model.PVC{
				{Name: "z-cache"},
				{Name: "a-data"},
			},
		}},
	}

	output := renderTestCluster(t, cluster)
	appsBlock := extractBlockFromMarker(t, output, fmt.Sprintf("  %s:", SanitizeID("apps")))

	assertAppearsInOrder(t, appsBlock,
		fmt.Sprintf("  %s -> %s: %s", SanitizeID("web"), PVCID("a-data"), strconv.Quote("/data (ro)")),
		fmt.Sprintf("  %s -> %s: %s", SanitizeID("web"), PVCID("z-cache"), strconv.Quote("/cache (rw)")),
	)
}

func TestRender_ProducesSameOutputForRepeatedRenders(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "beta",
			Services: []model.Service{
				{Name: "z-service", Type: "ClusterIP", Selector: map[string]string{"app": "api"}},
				{Name: "a-service", Type: "ClusterIP", Selector: map[string]string{"app": "api"}},
			},
			Deployments: []model.Workload{{
				Name:     "api",
				Kind:     "Deployment",
				Replicas: 1,
				Labels:   map[string]string{"app": "api"},
				VolumeMounts: []model.VolumeMount{
					{PVCName: "z-cache", MountPath: "/cache"},
					{PVCName: "a-data", MountPath: "/data"},
				},
			}},
			PVCs: []model.PVC{
				{Name: "z-cache"},
				{Name: "a-data"},
			},
		}, {
			Name: "alpha",
			DaemonSets: []model.Workload{{
				Name:     "node-agent",
				Kind:     "DaemonSet",
				Replicas: 2,
			}},
		}},
	}

	first := renderTestCluster(t, cluster)
	second := renderTestCluster(t, cluster)

	if first != second {
		t.Fatalf("expected repeated renders to match\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRender_EscapesIdentifiersAndLabels(t *testing.T) {
	cluster := &model.Cluster{
		Namespaces: []model.Namespace{{
			Name: "team.alpha",
			Deployments: []model.Workload{{
				Name:     "api.v2",
				Kind:     "Deployment",
				Replicas: 3,
				Labels: map[string]string{
					"app.kubernetes.io/name": "api.v2",
				},
			}},
			Services: []model.Service{{
				Name: "api.v2-service",
				Type: "ClusterIP",
				Ports: []model.Port{{
					Name:       "http",
					Port:       80,
					TargetPort: "api-http",
				}},
				Selector: map[string]string{
					"app.kubernetes.io/name": "api.v2",
				},
			}},
			PVCs: []model.PVC{{
				Name:         "cache.data",
				Capacity:     "10Gi",
				StorageClass: "fast.ssd",
			}},
		}},
	}

	output := renderTestCluster(t, cluster)

	if !strings.Contains(output, fmt.Sprintf("  %s: {", SanitizeID("team.alpha"))) {
		t.Fatalf("expected escaped namespace identifier, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("label: %s", strconv.Quote("team.alpha"))) {
		t.Fatalf("expected quoted namespace label, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("  %s: {", ServiceID("api.v2-service"))) {
		t.Fatalf("expected escaped service identifier, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("  %s: {", PVCID("cache.data"))) {
		t.Fatalf("expected escaped pvc identifier, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("label: %s", strconv.Quote("● api.v2 (3)"))) {
		t.Fatalf("expected quoted workload label, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("label: %s", strconv.Quote("⎈ api.v2-service\nClusterIP\nhttp: 80 -> api-http"))) {
		t.Fatalf("expected quoted service label with escaped newline, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("label: %s", strconv.Quote("💾 cache.data\n10Gi\n[fast.ssd]"))) {
		t.Fatalf("expected quoted pvc label with escaped newline, output was:\n%s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("%s -> %s", ServiceID("api.v2-service"), SanitizeID("api.v2"))) {
		t.Fatalf("expected escaped service-to-workload edge, output was:\n%s", output)
	}
}

func TestRender_IncludesServicePortsInLabels(t *testing.T) {
	cluster := &model.Cluster{Namespaces: []model.Namespace{{
		Name: "apps",
		Services: []model.Service{{
			Name: "api",
			Type: "ClusterIP",
			Ports: []model.Port{{
				Name:       "http",
				Port:       80,
				TargetPort: "web",
			}, {
				Name:       "metrics",
				Port:       9090,
				TargetPort: "9090",
			}},
		}},
	}}}

	output := renderTestCluster(t, cluster)
	if !strings.Contains(output, fmt.Sprintf("label: %s", strconv.Quote("⎈ api\nClusterIP\nhttp: 80 -> web\nmetrics: 9090"))) {
		t.Fatalf("expected service ports in label, output was:\n%s", output)
	}
}

func TestSanitizeID_AvoidsLossyCollisions(t *testing.T) {
	first := SanitizeID("api-v2")
	second := SanitizeID("api_v2")

	if first == second {
		t.Fatalf("expected distinct ids for different Kubernetes names, got %q", first)
	}
}

func TestQuoteString_EscapesQuotesAndNewlines(t *testing.T) {
	input := "name \"quoted\"\nnext"
	if got, want := QuoteString(input), strconv.Quote(input); got != want {
		t.Fatalf("QuoteString() = %q, want %q", got, want)
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

func assertAppearsInOrder(t *testing.T, input string, markers ...string) {
	t.Helper()

	lastIndex := -1
	for _, marker := range markers {
		index := strings.Index(input, marker)
		if index == -1 {
			t.Fatalf("missing marker %q in output:\n%s", marker, input)
		}
		if index <= lastIndex {
			t.Fatalf("expected %q to appear after previous marker in output:\n%s", marker, input)
		}
		lastIndex = index
	}
}
