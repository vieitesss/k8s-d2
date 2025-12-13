package model

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
	Name     string
	Kind     string // Deployment, StatefulSet, DaemonSet
	Replicas int32
	Labels   map[string]string
	PVCNames []string
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
