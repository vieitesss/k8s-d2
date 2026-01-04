package validation

import (
	"fmt"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/util"
)

// D2Validator validates D2 diagram output against expected topology
type D2Validator struct {
	expected *model.Cluster
	actual   string
	deriver  *RelationshipDeriver
}

// NewD2Validator creates a new validator with expected topology and actual D2 output
func NewD2Validator(expected *model.Cluster, d2Output string) *D2Validator {
	return &D2Validator{
		expected: expected,
		actual:   d2Output,
		deriver:  NewRelationshipDeriver(),
	}
}

// ValidateSyntax checks that the D2 output has valid syntax
func (v *D2Validator) ValidateSyntax() error {
	openBraces := strings.Count(v.actual, "{")
	closeBraces := strings.Count(v.actual, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	if !strings.Contains(v.actual, "direction:") {
		return fmt.Errorf("missing direction header")
	}

	return nil
}

// ValidateResources checks that all expected resources are present in the D2 output
func (v *D2Validator) ValidateResources() error {
	for _, ns := range v.expected.Namespaces {
		// Check namespace container
		nsID := util.SanitizeID(ns.Name)
		if !strings.Contains(v.actual, nsID) {
			return fmt.Errorf("missing namespace: %s", ns.Name)
		}

		// Check all workloads
		allWorkloads := []model.Workload{}
		allWorkloads = append(allWorkloads, ns.Deployments...)
		allWorkloads = append(allWorkloads, ns.StatefulSets...)
		allWorkloads = append(allWorkloads, ns.DaemonSets...)

		for _, w := range allWorkloads {
			wID := util.SanitizeID(w.Name)
			if !strings.Contains(v.actual, wID) {
				return fmt.Errorf("missing workload: %s (%s)", w.Name, w.Kind)
			}
		}

		// Check services
		for _, svc := range ns.Services {
			svcID := "svc_" + util.SanitizeID(svc.Name)
			if !strings.Contains(v.actual, svcID) {
				return fmt.Errorf("missing service: %s", svc.Name)
			}
		}

		// Check PVCs
		for _, pvc := range ns.PVCs {
			pvcID := "pvc_" + util.SanitizeID(pvc.Name)
			if !strings.Contains(v.actual, pvcID) {
				return fmt.Errorf("missing PVC: %s", pvc.Name)
			}
		}

		// Check config node if ConfigMaps or Secrets exist
		if ns.ConfigMaps > 0 || ns.Secrets > 0 {
			if !strings.Contains(v.actual, "_config") {
				return fmt.Errorf("missing config node for namespace: %s", ns.Name)
			}
		}
	}

	return nil
}

// ValidateWorkloadLabels checks that workload labels have correct icons and replica counts
func (v *D2Validator) ValidateWorkloadLabels() error {
	for _, ns := range v.expected.Namespaces {
		// Check deployments and statefulsets (they have replica counts)
		workloadsWithReplicas := []model.Workload{}
		workloadsWithReplicas = append(workloadsWithReplicas, ns.Deployments...)
		workloadsWithReplicas = append(workloadsWithReplicas, ns.StatefulSets...)

		for _, w := range workloadsWithReplicas {
			icon := util.WorkloadIcon(w.Kind)
			expectedLabel := fmt.Sprintf("%s %s (%d)", icon, w.Name, w.Replicas)
			if !strings.Contains(v.actual, expectedLabel) {
				return fmt.Errorf("incorrect label for %s (expected: %s)", w.Name, expectedLabel)
			}
		}

		// DaemonSets don't show replica count
		for _, w := range ns.DaemonSets {
			icon := util.WorkloadIcon(w.Kind)
			expectedLabel := fmt.Sprintf("%s %s", icon, w.Name)
			if !strings.Contains(v.actual, expectedLabel) {
				return fmt.Errorf("incorrect label for %s (expected: %s)", w.Name, expectedLabel)
			}
		}
	}

	return nil
}

// ValidateServiceConnections checks that service-to-workload connections exist
func (v *D2Validator) ValidateServiceConnections() error {
	for _, ns := range v.expected.Namespaces {
		connections := v.deriver.ServiceToWorkloadConnections(&ns)

		for _, conn := range connections {
			connectionStr := fmt.Sprintf("%s -> %s", conn.From, conn.To)
			if !strings.Contains(v.actual, connectionStr) {
				return fmt.Errorf("missing expected connection: %s", connectionStr)
			}
		}
	}

	return nil
}

// ValidatePVCConnections checks that workload-to-PVC connections exist
func (v *D2Validator) ValidatePVCConnections() error {
	for _, ns := range v.expected.Namespaces {
		// Only validate if there are PVCs
		if len(ns.PVCs) == 0 {
			continue
		}

		connections := v.deriver.WorkloadToPVCConnections(&ns)

		for _, conn := range connections {
			connectionStr := fmt.Sprintf("%s -> %s", conn.From, conn.To)
			if !strings.Contains(v.actual, connectionStr) {
				return fmt.Errorf("missing workload-to-PVC connection: %s", connectionStr)
			}
		}
	}

	return nil
}

// ValidateConfigInfo checks that ConfigMap/Secret counts match expected values
func (v *D2Validator) ValidateConfigInfo() error {
	for _, ns := range v.expected.Namespaces {
		if ns.ConfigMaps == 0 && ns.Secrets == 0 {
			continue
		}

		cmStr := fmt.Sprintf("CM: %d", ns.ConfigMaps)
		secStr := fmt.Sprintf("Sec: %d", ns.Secrets)

		if !strings.Contains(v.actual, cmStr) {
			return fmt.Errorf("incorrect ConfigMap count for namespace %s (expected: %d)", ns.Name, ns.ConfigMaps)
		}

		if !strings.Contains(v.actual, secStr) {
			return fmt.Errorf("incorrect Secret count for namespace %s (expected: %d)", ns.Name, ns.Secrets)
		}
	}

	return nil
}
