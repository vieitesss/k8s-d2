package model

import "testing"

func TestAllWorkloads_PreservesNamespaceWorkloadOrder(t *testing.T) {
	ns := &Namespace{
		Deployments:  []Workload{{Name: "api", Kind: "Deployment"}},
		StatefulSets: []Workload{{Name: "db", Kind: "StatefulSet"}},
		DaemonSets:   []Workload{{Name: "agent", Kind: "DaemonSet"}},
	}

	workloads := AllWorkloads(ns)
	if len(workloads) != 3 {
		t.Fatalf("expected 3 workloads, got %d", len(workloads))
	}

	if workloads[0].Name != "api" || workloads[0].Kind != "Deployment" {
		t.Fatalf("expected deployment first, got %+v", workloads[0])
	}
	if workloads[1].Name != "db" || workloads[1].Kind != "StatefulSet" {
		t.Fatalf("expected statefulset second, got %+v", workloads[1])
	}
	if workloads[2].Name != "agent" || workloads[2].Kind != "DaemonSet" {
		t.Fatalf("expected daemonset third, got %+v", workloads[2])
	}
}
