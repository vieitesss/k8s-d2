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
	Entrypoints  []Entrypoint
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

// Entrypoint represents traffic entering the cluster before it reaches a
// Service, such as an Ingress or an externally exposed Service type.
type Entrypoint struct {
	Name     string
	Kind     string // Ingress, NodePort, LoadBalancer
	Class    string
	Hosts    []string
	Services []string
	Ports    []Port
}

type Port struct {
	Name       string
	Port       int32
	TargetPort int32
	NodePort   int32
}

type PVC struct {
	Name         string
	StorageClass string
	Capacity     string
	BoundPod     string
}
