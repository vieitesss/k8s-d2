package model

// AllWorkloads returns namespace workloads in render order: Deployments,
// StatefulSets, then DaemonSets.
func AllWorkloads(ns *Namespace) []Workload {
	if ns == nil {
		return nil
	}

	workloads := make([]Workload, 0, len(ns.Deployments)+len(ns.StatefulSets)+len(ns.DaemonSets))
	workloads = append(workloads, ns.Deployments...)
	workloads = append(workloads, ns.StatefulSets...)
	workloads = append(workloads, ns.DaemonSets...)
	return workloads
}
