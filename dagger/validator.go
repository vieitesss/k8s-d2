package main

import (
	"fmt"
	"strings"
)

// D2Validator validates D2 diagram output
type D2Validator struct {
	output string
}

// NewD2Validator creates a new validator for the given D2 output
func NewD2Validator(output string) *D2Validator {
	return &D2Validator{output: output}
}

// ValidateSyntax checks that the D2 output has valid syntax
func (v *D2Validator) ValidateSyntax() error {
	openBraces := strings.Count(v.output, "{")
	closeBraces := strings.Count(v.output, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	if !strings.Contains(v.output, "direction:") {
		return fmt.Errorf("missing direction header")
	}

	return nil
}

// ValidateBasicTopology checks that expected resources are present in the output
func (v *D2Validator) ValidateBasicTopology() error {
	expectedResources := []string{
		"k8s_d2_test",          // Namespace container
		"web_frontend",         // Deployment
		"api_backend",          // Deployment
		"database",             // StatefulSet
		"log_collector",        // DaemonSet
		"svc_web_service",      // Service
		"svc_api_service",      // Service
		"svc_database_service", // Service
		"_config",              // ConfigMaps/Secrets node
	}

	for _, res := range expectedResources {
		if !strings.Contains(v.output, res) {
			return fmt.Errorf("missing expected resource: %s", res)
		}
	}

	return nil
}

// ValidateLabels checks that workload labels have correct icons and replica counts
func (v *D2Validator) ValidateLabels() error {
	expectedLabels := map[string]string{
		"web_frontend":  "● web-frontend (3)", // Deployment icon
		"database":      "◉ database (2)",     // StatefulSet icon
		"log_collector": "◈ log-collector",    // DaemonSet icon
	}

	for id, expectedLabel := range expectedLabels {
		if !strings.Contains(v.output, expectedLabel) {
			return fmt.Errorf("incorrect label for %s (expected: %s)", id, expectedLabel)
		}
	}

	// Check ConfigMap/Secret count
	if !strings.Contains(v.output, "CM: 2") || !strings.Contains(v.output, "Sec: 2") {
		return fmt.Errorf("incorrect ConfigMap/Secret counts (expected: CM: 2 | Sec: 2)")
	}

	return nil
}

// ValidateConnections checks that service-to-workload connections exist
func (v *D2Validator) ValidateConnections() error {
	expectedConnections := []string{
		"svc_web_service -> web_frontend",
		"svc_api_service -> api_backend",
		"svc_database_service -> database",
	}

	for _, conn := range expectedConnections {
		if !strings.Contains(v.output, conn) {
			return fmt.Errorf("missing expected connection: %s", conn)
		}
	}

	return nil
}

// ValidateStorage checks that PVC nodes are present (for --include-storage tests)
func (v *D2Validator) ValidateStorage() error {
	expectedStorage := []string{
		"data_volume", // PVC
		"logs_volume", // PVC
	}

	for _, res := range expectedStorage {
		if !strings.Contains(v.output, res) {
			return fmt.Errorf("missing expected storage resource: %s", res)
		}
	}

	return nil
}
