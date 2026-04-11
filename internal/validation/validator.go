package validation

import (
	"fmt"
	"sort"
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
	expectedEntries := render.LegendEntryIDs(v.expected)
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

	for _, entry := range render.AllLegendEntryIDs() {
		if _, ok := expectedSet[entry]; ok {
			continue
		}
		if strings.Contains(legendBlock, fmt.Sprintf("%s: {", entry)) {
			return fmt.Errorf("unexpected legend entry: %s", entry)
		}
	}

	return nil
}

// ValidateExactRender checks that the actual D2 output matches the current renderer output exactly.
func (v *D2Validator) ValidateExactRender() error {
	var expected strings.Builder
	renderer := render.NewD2Renderer(&expected, 0)
	if err := renderer.Render(v.expected); err != nil {
		return fmt.Errorf("render expected topology: %w", err)
	}

	expectedOutput := expected.String()
	if expectedOutput != v.actual {
		return exactRenderMismatch(expectedOutput, v.actual)
	}

	return nil
}

// ValidateResources checks that all expected resources are present in the D2 output
func (v *D2Validator) ValidateResources() error {
	for _, ns := range v.expected.Namespaces {
		namespaceBlock, err := extractNamespaceBlock(v.actual, ns.Name)
		if err != nil {
			return err
		}

		nsID := render.SanitizeID(ns.Name)
		if !containsD2Line(namespaceBlock, fmt.Sprintf("%s: {", nsID)) {
			return fmt.Errorf("missing namespace: %s", ns.Name)
		}
		if !containsD2Line(namespaceBlock, fmt.Sprintf("label: %s", render.QuoteString(ns.Name))) {
			return fmt.Errorf("incorrect namespace label for %s", ns.Name)
		}

		allWorkloads := []model.Workload{}
		allWorkloads = append(allWorkloads, ns.Deployments...)
		allWorkloads = append(allWorkloads, ns.StatefulSets...)
		allWorkloads = append(allWorkloads, ns.DaemonSets...)

		for _, w := range allWorkloads {
			wID := render.SanitizeID(w.Name)
			if !containsD2Line(namespaceBlock, fmt.Sprintf("%s: {", wID)) {
				return fmt.Errorf("missing workload: %s (%s)", w.Name, w.Kind)
			}
		}

		for _, svc := range ns.Services {
			svcID := render.ServiceID(svc.Name)
			if !containsD2Line(namespaceBlock, fmt.Sprintf("%s: {", svcID)) {
				return fmt.Errorf("missing service: %s", svc.Name)
			}
		}

		for _, pvc := range ns.PVCs {
			pvcID := render.PVCID(pvc.Name)
			if !containsD2Line(namespaceBlock, fmt.Sprintf("%s: {", pvcID)) {
				return fmt.Errorf("missing PVC: %s", pvc.Name)
			}
		}

		if ns.ConfigMaps > 0 || ns.Secrets > 0 {
			if !containsD2Line(namespaceBlock, "_config: {") {
				return fmt.Errorf("missing config node for namespace: %s", ns.Name)
			}
		}
	}

	return nil
}

// ValidateWorkloadLabels checks that workload labels have correct icons and replica counts
func (v *D2Validator) ValidateWorkloadLabels() error {
	for _, ns := range v.expected.Namespaces {
		namespaceBlock, err := extractNamespaceBlock(v.actual, ns.Name)
		if err != nil {
			return err
		}

		// Check deployments and statefulsets (they have replica counts)
		workloadsWithReplicas := []model.Workload{}
		workloadsWithReplicas = append(workloadsWithReplicas, ns.Deployments...)
		workloadsWithReplicas = append(workloadsWithReplicas, ns.StatefulSets...)

		for _, w := range workloadsWithReplicas {
			expectedLabel := render.WorkloadLabel(w)
			if !containsD2Line(namespaceBlock, fmt.Sprintf("label: %s", render.QuoteString(expectedLabel))) {
				return fmt.Errorf("incorrect label for %s (expected: %s)", w.Name, expectedLabel)
			}
		}

		for _, w := range ns.DaemonSets {
			expectedLabel := render.WorkloadLabel(w)
			if !containsD2Line(namespaceBlock, fmt.Sprintf("label: %s", render.QuoteString(expectedLabel))) {
				return fmt.Errorf("incorrect label for %s (expected: %s)", w.Name, expectedLabel)
			}
		}
	}

	return nil
}

// ValidateServiceConnections checks that service-to-workload connections exist
func (v *D2Validator) ValidateServiceConnections() error {
	for _, ns := range v.expected.Namespaces {
		namespaceBlock, err := extractNamespaceBlock(v.actual, ns.Name)
		if err != nil {
			return err
		}

		connections := v.deriver.ServiceToWorkloadConnections(&ns)
		sortConnections(connections)

		for _, conn := range connections {
			connectionStr := fmt.Sprintf("%s -> %s", conn.From, conn.To)
			if !containsD2Line(namespaceBlock, connectionStr) {
				return fmt.Errorf("missing expected connection: %s -> %s", conn.From, conn.To)
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

		namespaceBlock, err := extractNamespaceBlock(v.actual, ns.Name)
		if err != nil {
			return err
		}

		connections := v.deriver.WorkloadToPVCConnections(&ns)
		sortConnections(connections)

		for _, conn := range connections {
			baseConnectionStr := fmt.Sprintf("%s -> %s", conn.From, conn.To)
			if !containsD2LinePrefix(namespaceBlock, baseConnectionStr) {
				return fmt.Errorf("missing workload-to-PVC connection: %s -> %s", conn.From, conn.To)
			}

			if conn.Label != "" {
				fullConnectionStr := fmt.Sprintf("%s: %s", baseConnectionStr, render.QuoteString(conn.Label))
				if !containsD2Line(namespaceBlock, fullConnectionStr) {
					return fmt.Errorf(
						"connection %s missing expected mount metadata: %s",
						fmt.Sprintf("%s -> %s", conn.From, conn.To),
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

		namespaceBlock, err := extractNamespaceBlock(v.actual, ns.Name)
		if err != nil {
			return err
		}

		expectedLabel := fmt.Sprintf("label: %s", render.QuoteString(fmt.Sprintf("CM: %d | Sec: %d", ns.ConfigMaps, ns.Secrets)))

		if !containsD2Line(namespaceBlock, expectedLabel) {
			return fmt.Errorf("incorrect config info for namespace %s (expected label: %s)", ns.Name, expectedLabel)
		}
	}

	return nil
}

func exactRenderMismatch(expected, actual string) error {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	lineCount := len(expectedLines)
	if len(actualLines) > lineCount {
		lineCount = len(actualLines)
	}

	for i := 0; i < lineCount; i++ {
		expectedLine := "<missing>"
		actualLine := "<missing>"
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}
		if expectedLine != actualLine {
			return fmt.Errorf(
				"actual D2 output does not match exact renderer output: first difference at line %d (expected %q, actual %q)",
				i+1,
				expectedLine,
				actualLine,
			)
		}
	}

	return fmt.Errorf(
		"actual D2 output does not match exact renderer output: expected %d bytes, actual %d bytes",
		len(expected),
		len(actual),
	)
}

func sortConnections(connections []Connection) {
	sort.Slice(connections, func(i, j int) bool {
		if connections[i].From != connections[j].From {
			return connections[i].From < connections[j].From
		}
		if connections[i].To != connections[j].To {
			return connections[i].To < connections[j].To
		}
		return connections[i].Label < connections[j].Label
	})
}

func containsD2Line(input, expected string) bool {
	for _, line := range strings.Split(input, "\n") {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}

	return false
}

func containsD2LinePrefix(input, prefix string) bool {
	for _, line := range strings.Split(input, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}

	return false
}

func extractNamespaceBlock(input, namespaceName string) (string, error) {
	nsID := render.SanitizeID(namespaceName)
	block, err := extractD2Block(input, fmt.Sprintf("  %s: {", nsID))
	if err != nil {
		if strings.Contains(err.Error(), "missing") {
			return "", fmt.Errorf("missing namespace: %s", namespaceName)
		}
		return "", fmt.Errorf("invalid namespace block for %s: %w", namespaceName, err)
	}

	return block, nil
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
