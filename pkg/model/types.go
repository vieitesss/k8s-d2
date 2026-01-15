package model

import (
	"fmt"
	"strings"
)

type Cluster struct {
	Name       string
	Namespaces []Namespace
}

type Namespace struct {
	Name         string
	Deployments  []Workload
	StatefulSets []Workload
	DaemonSets   []Workload
	Services     []Service
	ConfigMaps   int
	Secrets      int
	PVCs         []PVC
}

type Workload struct {
	Name         string
	Kind         string // Deployment, StatefulSet, DaemonSet
	Replicas     int32
	Labels       map[string]string
	VolumeMounts []VolumeMount
}

// VolumeMount represents a volume mount in a workload container, capturing
// the PVC reference and mount metadata (path, read-only status).
type VolumeMount struct {
	PVCName   string // Name of the PersistentVolumeClaim
	MountPath string // Path where volume is mounted (e.g., "/var/log/app")
	ReadOnly  bool   // Whether volume is mounted read-only
}

type Service struct {
	Name     string
	Type     string // ClusterIP, NodePort, LoadBalancer
	Selector map[string]string
	Ports    []Port
}

type Port struct {
	Name       string
	Port       int32
	TargetPort int32
}

type PVC struct {
	Name         string
	StorageClass string
	Capacity     string
	BoundPod     string
}

// FormatMountLabel creates a compact label for volume mounts.
// Single mount: "/var/log/app (rw)"
// Multiple mounts: "/data (rw)\\n/backup (ro)"
// Note: Uses \\n for D2 newlines in labels.
func FormatMountLabel(mounts []VolumeMount) string {
	labels := make([]string, len(mounts))
	for i, m := range mounts {
		accessMode := "rw"
		if m.ReadOnly {
			accessMode = "ro"
		}
		labels[i] = fmt.Sprintf("%s (%s)", m.MountPath, accessMode)
	}
	return strings.Join(labels, "\\n")
}
