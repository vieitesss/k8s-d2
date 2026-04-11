package model

import (
	"fmt"
	"strings"
)

// FormatMountLabel creates a compact label for volume mounts.
// Single mount: "/var/log/app (rw)"
// Multiple mounts: "/data (rw)\n/backup (ro)"
func FormatMountLabel(mounts []VolumeMount) string {
	labels := make([]string, len(mounts))
	for i, m := range mounts {
		accessMode := "rw"
		if m.ReadOnly {
			accessMode = "ro"
		}
		labels[i] = fmt.Sprintf("%s (%s)", m.MountPath, accessMode)
	}
	return strings.Join(labels, "\n")
}

// FormatServicePortLabel creates a compact label for a single service port.
// Examples: "80", "http: 80 -> web", "metrics: 9090 -> 9091".
func FormatServicePortLabel(port Port) string {
	portValue := fmt.Sprintf("%d", port.Port)
	label := portValue
	if port.Name != "" {
		label = fmt.Sprintf("%s: %s", port.Name, portValue)
	}
	if port.TargetPort != "" && port.TargetPort != portValue {
		label = fmt.Sprintf("%s -> %s", label, port.TargetPort)
	}
	return label
}

// FormatServicePortsLabel joins multiple service ports in render order.
func FormatServicePortsLabel(ports []Port) string {
	labels := make([]string, len(ports))
	for i, port := range ports {
		labels[i] = FormatServicePortLabel(port)
	}
	return strings.Join(labels, "\n")
}
