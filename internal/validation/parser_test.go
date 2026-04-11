package validation_test

import (
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/model"
)

func TestFixtureParser_ParseServicePreservesNamedTargetPort(t *testing.T) {
	parser := validation.NewFixtureParser("apps")
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
