package validation_test

import (
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/model"
)

func TestFixtureParser_ParseServicePreservesNamedTargetPort(t *testing.T) {
	parser := validation.NewFixtureParser("apps", false)
	cluster, err := parser.ParseFixtures([][]byte{[]byte(`apiVersion: v1
kind: Service
metadata:
  name: api
  namespace: apps
spec:
  type: ClusterIP
  selector:
    app: api
  ports:
  - name: http
    port: 80
    targetPort: web
  - port: 443
`)})
	if err != nil {
		t.Fatalf("ParseFixtures returned error: %v", err)
	}

	got := cluster.Namespaces[0].Services[0].Ports
	want := []model.Port{{Name: "http", Port: 80, TargetPort: "web"}, {Port: 443, TargetPort: "443"}}
	if len(got) != len(want) {
		t.Fatalf("expected %d ports, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("port %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFixtureParser_AllowsWorkloadsWithoutSelectors(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		assert  func(*testing.T, model.Namespace)
	}{
		{
			name: "deployment without selector",
			fixture: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: apps
spec:
  template:
    spec:
      containers:
      - name: api
        image: registry.k8s.io/pause:3.10
`,
			assert: func(t *testing.T, ns model.Namespace) {
				t.Helper()
				if len(ns.Deployments) != 1 {
					t.Fatalf("expected 1 deployment, got %d", len(ns.Deployments))
				}
				if len(ns.Deployments[0].Labels) != 0 {
					t.Fatalf("expected empty deployment labels, got %+v", ns.Deployments[0].Labels)
				}
			},
		},
		{
			name: "statefulset without selector",
			fixture: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
  namespace: apps
spec:
  serviceName: db
  template:
    spec:
      containers:
      - name: db
        image: registry.k8s.io/pause:3.10
`,
			assert: func(t *testing.T, ns model.Namespace) {
				t.Helper()
				if len(ns.StatefulSets) != 1 {
					t.Fatalf("expected 1 statefulset, got %d", len(ns.StatefulSets))
				}
				if len(ns.StatefulSets[0].Labels) != 0 {
					t.Fatalf("expected empty statefulset labels, got %+v", ns.StatefulSets[0].Labels)
				}
			},
		},
		{
			name: "daemonset without selector",
			fixture: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-agent
  namespace: apps
spec:
  template:
    spec:
      containers:
      - name: agent
        image: registry.k8s.io/pause:3.10
`,
			assert: func(t *testing.T, ns model.Namespace) {
				t.Helper()
				if len(ns.DaemonSets) != 1 {
					t.Fatalf("expected 1 daemonset, got %d", len(ns.DaemonSets))
				}
				if len(ns.DaemonSets[0].Labels) != 0 {
					t.Fatalf("expected empty daemonset labels, got %+v", ns.DaemonSets[0].Labels)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := validation.NewFixtureParser("apps", false)
			cluster, err := parser.ParseFixtures([][]byte{[]byte(tt.fixture)})
			if err != nil {
				t.Fatalf("ParseFixtures returned error: %v", err)
			}
			if len(cluster.Namespaces) != 1 {
				t.Fatalf("expected 1 namespace, got %d", len(cluster.Namespaces))
			}

			tt.assert(t, cluster.Namespaces[0])
		})
	}
}
