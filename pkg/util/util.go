package util

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// SanitizeID converts a Kubernetes resource name to a valid D2 identifier.
// D2 syntax doesn't allow hyphens, so we convert them to underscores.
func SanitizeID(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

// WorkloadIcon returns the D2 icon for a workload type.
func WorkloadIcon(kind string) string {
	switch kind {
	case "StatefulSet":
		return "◉"
	case "DaemonSet":
		return "◈"
	default:
		return "●"
	}
}

// LabelsMatch checks if a selector matches a set of labels.
// All selector key-value pairs must match the labels for this to return true.
func LabelsMatch(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// ExtractPVCNames extracts PVC names from a pod's volumes slice.
func ExtractPVCNames(volumes []corev1.Volume) []string {
	var pvcNames []string
	for _, vol := range volumes {
		if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName != "" {
			pvcNames = append(pvcNames, vol.PersistentVolumeClaim.ClaimName)
		}
	}
	return pvcNames
}

// ExtractAllStatefulSetPVCNames extracts all PVC names from a StatefulSet, including
// both regular pod volumes and generated names from volumeClaimTemplates.
// StatefulSet creates PVCs with pattern: <templateName>-<statefulsetName>-<ordinal>
func ExtractAllStatefulSetPVCNames(volumes []corev1.Volume, templates []corev1.PersistentVolumeClaim, ssName string, replicas int32) []string {
	// Start with regular pod volumes
	pvcNames := ExtractPVCNames(volumes)

	// Add generated names from volumeClaimTemplates
	for _, vct := range templates {
		for i := range replicas {
			pvcName := fmt.Sprintf("%s-%s-%d", vct.Name, ssName, i)
			pvcNames = append(pvcNames, pvcName)
		}
	}

	return pvcNames
}
