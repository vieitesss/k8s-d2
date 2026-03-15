package validation

import (
	"fmt"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
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

// ValidateSyntax checks that the D2 output has valid syntax.
// Only validates braces and direction header since the rendering logic is deterministic
// and content validation is handled by other validators (resources, connections, etc.)
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

// ValidateLegendStructure checks that the built-in D2 legend exists when needed
// and only includes entries for rendered resource types.
func (v *D2Validator) ValidateLegendStructure() error {
	expectedEntries := v.expectedLegendEntries()
	if len(expectedEntries) == 0 {
		if strings.Contains(v.actual, "d2-legend:") {
			return fmt.Errorf("unexpected legend block for topology without legend entries")
		}
		return nil
	}

	if !strings.Contains(v.actual, "vars: {") {
		return fmt.Errorf("missing vars block for legend")
	}

	legendBlock, err := extractD2Block(v.actual, "d2-legend:")
	if err != nil {
		return err
	}

	expectedSet := make(map[string]struct{}, len(expectedEntries))
	for _, entry := range expectedEntries {
		expectedSet[entry] = struct{}{}
		if !strings.Contains(legendBlock, fmt.Sprintf("%s: {", entry)) {
			return fmt.Errorf("missing legend entry: %s", entry)
		}
	}

	for _, entry := range allLegendEntryIDs() {
		if _, ok := expectedSet[entry]; ok {
			continue
		}
		if strings.Contains(legendBlock, fmt.Sprintf("%s: {", entry)) {
			return fmt.Errorf("unexpected legend entry: %s", entry)
		}
	}

	return nil
}

// ValidateResources checks that all expected resources are present in the D2 output
func (v *D2Validator) ValidateResources() error {
	for _, ns := range v.expected.Namespaces {
		// Check namespace container
		nsID := render.SanitizeID(ns.Name)
		if !strings.Contains(v.actual, nsID) {
			return fmt.Errorf("missing namespace: %s", ns.Name)
		}

		// Check all workloads
		allWorkloads := []model.Workload{}
		allWorkloads = append(allWorkloads, ns.Deployments...)
		allWorkloads = append(allWorkloads, ns.StatefulSets...)
		allWorkloads = append(allWorkloads, ns.DaemonSets...)

		for _, w := range allWorkloads {
			wID := render.SanitizeID(w.Name)
			if !strings.Contains(v.actual, wID) {
				return fmt.Errorf("missing workload: %s (%s)", w.Name, w.Kind)
			}
		}

		// Check services
		for _, svc := range ns.Services {
			svcID := "svc_" + render.SanitizeID(svc.Name)
			if !strings.Contains(v.actual, svcID) {
				return fmt.Errorf("missing service: %s", svc.Name)
			}
		}

		// Check PVCs
		for _, pvc := range ns.PVCs {
			pvcID := "pvc_" + render.SanitizeID(pvc.Name)
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
			icon := render.WorkloadIcon(w.Kind)
			expectedLabel := fmt.Sprintf("%s %s (%d)", icon, w.Name, w.Replicas)
			if !strings.Contains(v.actual, expectedLabel) {
				return fmt.Errorf("incorrect label for %s (expected: %s)", w.Name, expectedLabel)
			}
		}

		// DaemonSets don't show replica count
		for _, w := range ns.DaemonSets {
			icon := render.WorkloadIcon(w.Kind)
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

// ValidatePVCConnections checks that workload-to-PVC connections exist with mount metadata
func (v *D2Validator) ValidatePVCConnections() error {
	for _, ns := range v.expected.Namespaces {
		// Only validate if there are PVCs
		if len(ns.PVCs) == 0 {
			continue
		}

		connections := v.deriver.WorkloadToPVCConnections(&ns)

		for _, conn := range connections {
			// Check basic connection exists
			baseConnectionStr := fmt.Sprintf("%s -> %s", conn.From, conn.To)
			if !strings.Contains(v.actual, baseConnectionStr) {
				return fmt.Errorf("missing workload-to-PVC connection: %s", baseConnectionStr)
			}

			// If connection has mount metadata, validate the label appears
			if conn.Label != "" {
				fullConnectionStr := fmt.Sprintf("%s: \"%s\"", baseConnectionStr, conn.Label)
				if !strings.Contains(v.actual, fullConnectionStr) {
					return fmt.Errorf(
						"connection %s missing expected mount metadata: %s",
						baseConnectionStr,
						conn.Label,
					)
				}
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

func (v *D2Validator) expectedLegendEntries() []string {
	var hasDeployments bool
	var hasStatefulSets bool
	var hasDaemonSets bool
	var hasServices bool
	var hasConfig bool
	var hasPVCs bool

	for _, ns := range v.expected.Namespaces {
		hasDeployments = hasDeployments || len(ns.Deployments) > 0
		hasStatefulSets = hasStatefulSets || len(ns.StatefulSets) > 0
		hasDaemonSets = hasDaemonSets || len(ns.DaemonSets) > 0
		hasServices = hasServices || len(ns.Services) > 0
		hasConfig = hasConfig || ns.ConfigMaps > 0 || ns.Secrets > 0
		hasPVCs = hasPVCs || len(ns.PVCs) > 0
	}

	entries := make([]string, 0, 6)
	if hasDeployments {
		entries = append(entries, "deployment")
	}
	if hasStatefulSets {
		entries = append(entries, "statefulset")
	}
	if hasDaemonSets {
		entries = append(entries, "daemonset")
	}
	if hasServices {
		entries = append(entries, "service")
	}
	if hasConfig {
		entries = append(entries, "config")
	}
	if hasPVCs {
		entries = append(entries, "pvc")
	}

	return entries
}

func allLegendEntryIDs() []string {
	return []string{"deployment", "statefulset", "daemonset", "service", "config", "pvc"}
}

func extractD2Block(input, marker string) (string, error) {
	markerIndex := strings.Index(input, marker)
	if markerIndex == -1 {
		return "", fmt.Errorf("missing %s block", marker)
	}

	blockStart := strings.Index(input[markerIndex:], "{")
	if blockStart == -1 {
		return "", fmt.Errorf("missing opening brace for %s block", marker)
	}

	braceDepth := 0
	for i, r := range input[markerIndex+blockStart:] {
		switch r {
		case '{':
			braceDepth++
		case '}':
			braceDepth--
			if braceDepth == 0 {
				return input[markerIndex : markerIndex+blockStart+i+1], nil
			}
		}
	}

	return "", fmt.Errorf("unclosed %s block", marker)
}
