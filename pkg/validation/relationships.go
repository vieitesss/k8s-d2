package validation

import (
	"fmt"

	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/util"
)

// Connection represents a relationship between two resources in the D2 diagram
type Connection struct {
	From string // Source resource ID (e.g., "svc_web_service")
	To   string // Target resource ID (e.g., "web_frontend")
	Type string // Connection type: "service-to-workload" or "workload-to-pvc"
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
		svcID := util.SanitizeID(svc.Name)
		for _, w := range allWorkloads {
			if util.LabelsMatch(svc.Selector, w.Labels) {
				wID := util.SanitizeID(w.Name)
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

// WorkloadToPVCConnections derives all workload→PVC connections in a namespace
func (rd *RelationshipDeriver) WorkloadToPVCConnections(ns *model.Namespace) []Connection {
	var connections []Connection

	// Get all workloads in the namespace
	allWorkloads := []model.Workload{}
	allWorkloads = append(allWorkloads, ns.Deployments...)
	allWorkloads = append(allWorkloads, ns.StatefulSets...)
	allWorkloads = append(allWorkloads, ns.DaemonSets...)

	for _, w := range allWorkloads {
		wID := util.SanitizeID(w.Name)
		for _, pvcName := range w.PVCNames {
			pvcID := util.SanitizeID(pvcName)
			connections = append(connections, Connection{
				From: wID,
				To:   fmt.Sprintf("pvc_%s", pvcID),
				Type: "workload-to-pvc",
			})
		}
	}

	return connections
}
