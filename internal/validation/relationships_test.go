package validation_test

import (
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

func TestRelationshipDeriver_UsesKindAwareWorkloadIDsForSameNameWorkloads(t *testing.T) {
	ns := &model.Namespace{
		Name: "apps",
		Deployments: []model.Workload{{
			Name:   "api",
			Kind:   "Deployment",
			Labels: map[string]string{"app": "api"},
			VolumeMounts: []model.VolumeMount{{
				PVCName:   "deploy-data",
				MountPath: "/srv/api",
			}},
		}},
		StatefulSets: []model.Workload{{
			Name:   "api",
			Kind:   "StatefulSet",
			Labels: map[string]string{"app": "api"},
			VolumeMounts: []model.VolumeMount{{
				PVCName:   "state-data",
				MountPath: "/var/lib/api",
			}},
		}},
		Services: []model.Service{{
			Name:     "api-service",
			Type:     "ClusterIP",
			Selector: map[string]string{"app": "api"},
		}},
	}

	deriver := validation.NewRelationshipDeriver()

	serviceConnections := deriver.ServiceToWorkloadConnections(ns)
	if len(serviceConnections) != 2 {
		t.Fatalf("expected 2 service-to-workload connections, got %d", len(serviceConnections))
	}

	wantServiceTargets := map[string]struct{}{
		render.WorkloadID("Deployment", "api"):  {},
		render.WorkloadID("StatefulSet", "api"): {},
	}
	for _, conn := range serviceConnections {
		if conn.From != render.ServiceID("api-service") {
			t.Fatalf("unexpected service connection source %q", conn.From)
		}
		if _, ok := wantServiceTargets[conn.To]; !ok {
			t.Fatalf("unexpected service connection target %q", conn.To)
		}
		delete(wantServiceTargets, conn.To)
	}
	if len(wantServiceTargets) != 0 {
		t.Fatalf("missing service connection targets: %v", wantServiceTargets)
	}

	pvcConnections := deriver.WorkloadToPVCConnections(ns)
	if len(pvcConnections) != 2 {
		t.Fatalf("expected 2 workload-to-pvc connections, got %d", len(pvcConnections))
	}

	wantPVCConnections := map[string]struct {
		to    string
		label string
	}{
		render.WorkloadID("Deployment", "api"): {
			to:    render.PVCID("deploy-data"),
			label: "/srv/api (rw)",
		},
		render.WorkloadID("StatefulSet", "api"): {
			to:    render.PVCID("state-data"),
			label: "/var/lib/api (rw)",
		},
	}
	for _, conn := range pvcConnections {
		want, ok := wantPVCConnections[conn.From]
		if !ok {
			t.Fatalf("unexpected pvc connection source %q", conn.From)
		}
		if conn.To != want.to {
			t.Fatalf("unexpected pvc connection target for %q: got %q want %q", conn.From, conn.To, want.to)
		}
		if conn.Label != want.label {
			t.Fatalf("unexpected pvc connection label for %q: got %q want %q", conn.From, conn.Label, want.label)
		}
		delete(wantPVCConnections, conn.From)
	}
	if len(wantPVCConnections) != 0 {
		t.Fatalf("missing pvc connections: %v", wantPVCConnections)
	}
}
