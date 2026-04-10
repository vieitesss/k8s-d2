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
