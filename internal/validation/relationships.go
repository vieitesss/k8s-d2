package validation

import (
	"fmt"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

// Connection represents a relationship between two resources in the D2 diagram
type Connection struct {
	From  string // Source resource ID (e.g., "svc_web_service")
	To    string // Target resource ID (e.g., "web_frontend")
	Type  string // Connection type: "service-to-workload" or "workload-to-pvc"
	Label string // Connection label for mount metadata (e.g., "/var/log (rw)")
}

// RelationshipDeriver handles deriving connections between resources.
type RelationshipDeriver struct{}

// NewRelationshipDeriver creates a new RelationshipDeriver
func NewRelationshipDeriver() *RelationshipDeriver {
	return &RelationshipDeriver{}
}

// ServiceToWorkloadConnections derives all service→workload connections in a namespace
// based on label selector matching
func (rd *RelationshipDeriver) ServiceToWorkloadConnections(ns *model.Namespace) []Connection {
	var connections []Connection

	// Get all workloads in the namespace
	allWorkloads := []model.Workload{}
	allWorkloads = append(allWorkloads, ns.Deployments...)
	allWorkloads = append(allWorkloads, ns.StatefulSets...)
	allWorkloads = append(allWorkloads, ns.DaemonSets...)

	for _, svc := range ns.Services {
		svcID := render.SanitizeID(svc.Name)
		for _, w := range allWorkloads {
			if render.LabelsMatch(svc.Selector, w.Labels) {
				wID := render.SanitizeID(w.Name)
				connections = append(connections, Connection{
					From: fmt.Sprintf("svc_%s", svcID),
					To:   wID,
					Type: "service-to-workload",
				})
			}
		}
	}

	return connections
}

// formatMountLabel creates mount label for validation matching.
func formatMountLabel(mounts []model.VolumeMount) string {
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

// WorkloadToPVCConnections derives all workload→PVC connections in a namespace
func (rd *RelationshipDeriver) WorkloadToPVCConnections(ns *model.Namespace) []Connection {
	var connections []Connection

	// Get all workloads in the namespace
	allWorkloads := []model.Workload{}
	allWorkloads = append(allWorkloads, ns.Deployments...)
	allWorkloads = append(allWorkloads, ns.StatefulSets...)
	allWorkloads = append(allWorkloads, ns.DaemonSets...)

	for _, w := range allWorkloads {
		wID := render.SanitizeID(w.Name)

		// Group by PVC name (handle same PVC mounted at multiple paths)
		mountsByPVC := make(map[string][]model.VolumeMount)
		for _, mount := range w.VolumeMounts {
			mountsByPVC[mount.PVCName] = append(mountsByPVC[mount.PVCName], mount)
		}

		for pvcName, mounts := range mountsByPVC {
			pvcID := render.SanitizeID(pvcName)
			connections = append(connections, Connection{
				From:  wID,
				To:    fmt.Sprintf("pvc_%s", pvcID),
				Type:  "workload-to-pvc",
				Label: formatMountLabel(mounts),
			})
		}
	}

	return connections
}
