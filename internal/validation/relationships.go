package validation

import (
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
)

// Connection represents a relationship between two resources in the D2 diagram
type Connection struct {
	From  string // Source resource ID (e.g., "svc_id_7765622d73657276696365")
	To    string // Target resource ID (e.g., "id_7765622d66726f6e74656e64")
	Type  string // Connection type: "entrypoint-to-service", "service-to-workload", or "workload-to-pvc"
	Label string // Connection label for mount metadata (e.g., "/var/log (rw)")
}

// RelationshipDeriver handles deriving connections between resources.
type RelationshipDeriver struct{}

// NewRelationshipDeriver creates a new RelationshipDeriver
func NewRelationshipDeriver() *RelationshipDeriver {
	return &RelationshipDeriver{}
}

// EntrypointToServiceConnections derives all entrypoint→service connections in a
// namespace based on the backend services each entrypoint references.
func (rd *RelationshipDeriver) EntrypointToServiceConnections(ns *model.Namespace) []Connection {
	var connections []Connection

	services := make(map[string]struct{}, len(ns.Services))
	for _, svc := range ns.Services {
		services[svc.Name] = struct{}{}
	}

	for _, entrypoint := range ns.Entrypoints {
		entrypointID := render.EntrypointID(entrypoint.Kind, entrypoint.Name)
		for _, serviceName := range entrypoint.Services {
			if _, ok := services[serviceName]; !ok {
				continue
			}

			connections = append(connections, Connection{
				From: entrypointID,
				To:   render.ServiceID(serviceName),
				Type: "entrypoint-to-service",
			})
		}
	}

	return connections
}

// ServiceToWorkloadConnections derives all service→workload connections in a namespace
// based on label selector matching
func (rd *RelationshipDeriver) ServiceToWorkloadConnections(ns *model.Namespace) []Connection {
	var connections []Connection
	workloads := model.AllWorkloads(ns)

	for _, svc := range ns.Services {
		svcID := render.ServiceID(svc.Name)
		for _, w := range workloads {
			if render.LabelsMatch(svc.Selector, w.Labels) {
				connections = append(connections, Connection{
					From: svcID,
					To:   render.WorkloadID(w),
					Type: "service-to-workload",
				})
			}
		}
	}

	return connections
}

// WorkloadToPVCConnections derives all workload→PVC connections in a namespace
func (rd *RelationshipDeriver) WorkloadToPVCConnections(ns *model.Namespace) []Connection {
	var connections []Connection

	for _, w := range model.AllWorkloads(ns) {
		wID := render.WorkloadID(w)

		// Group by PVC name (handle same PVC mounted at multiple paths)
		mountsByPVC := make(map[string][]model.VolumeMount)
		for _, mount := range w.VolumeMounts {
			mountsByPVC[mount.PVCName] = append(mountsByPVC[mount.PVCName], mount)
		}

		for pvcName, mounts := range mountsByPVC {
			pvcID := render.PVCID(pvcName)
			connections = append(connections, Connection{
				From:  wID,
				To:    pvcID,
				Type:  "workload-to-pvc",
				Label: model.FormatMountLabel(mounts),
			})
		}
	}

	return connections
}
