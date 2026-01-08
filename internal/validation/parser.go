package validation

import (
	"bytes"

	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// FixtureParser parses Kubernetes YAML fixtures into internal model types
type FixtureParser struct {
	namespace string
}

// NewFixtureParser creates a new FixtureParser for the given namespace
func NewFixtureParser(namespace string) *FixtureParser {
	return &FixtureParser{namespace: namespace}
}

// ParseFixtures reads multiple YAML fixture files and builds a Cluster model.
// Each element in fixtureData represents the raw bytes of a YAML file (which may
// contain multiple resources separated by ---)
func (p *FixtureParser) ParseFixtures(fixtureData [][]byte) (*model.Cluster, error) {
	cluster := &model.Cluster{Name: "cluster"}
	ns := &model.Namespace{Name: p.namespace}

	for _, data := range fixtureData {
		if err := p.parseYAMLFile(data, ns); err != nil {
			return nil, err
		}
	}

	cluster.Namespaces = append(cluster.Namespaces, *ns)
	return cluster, nil
}

// parseYAMLFile handles a single YAML file that may contain multiple documents
func (p *FixtureParser) parseYAMLFile(data []byte, ns *model.Namespace) error {
	// Split multi-document YAML by --- separator
	docs := bytes.SplitSeq(data, []byte("\n---\n"))

	for doc := range docs {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		if err := p.parseYAMLDocument(doc, ns); err != nil {
			// Skip documents we can't parse (might be comments or invalid YAML)
			continue
		}
	}

	return nil
}

// parseYAMLDocument parses a single Kubernetes resource document
func (p *FixtureParser) parseYAMLDocument(doc []byte, ns *model.Namespace) error {
	// First, unmarshal to determine the kind
	var typeMeta struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}

	if err := yaml.Unmarshal(doc, &typeMeta); err != nil {
		return err
	}

	// Parse based on kind
	switch typeMeta.Kind {
	case "Deployment":
		return p.parseDeployment(doc, ns)
	case "StatefulSet":
		return p.parseStatefulSet(doc, ns)
	case "DaemonSet":
		return p.parseDaemonSet(doc, ns)
	case "Service":
		return p.parseService(doc, ns)
	case "PersistentVolumeClaim":
		return p.parsePVC(doc, ns)
	case "ConfigMap":
		return p.parseConfigMap(doc, ns)
	case "Secret":
		return p.parseSecret(doc, ns)
	case "Namespace", "StorageClass":
		// These don't need to be parsed into the model for validation
		return nil
	default:
		// Unknown kind, skip
		return nil
	}
}

// parseDeployment converts a Kubernetes Deployment to a model.Workload
func (p *FixtureParser) parseDeployment(doc []byte, ns *model.Namespace) error {
	var dep appsv1.Deployment
	if err := yaml.Unmarshal(doc, &dep); err != nil {
		return err
	}

	replicas := int32(1) // Default replica count
	if dep.Spec.Replicas != nil {
		replicas = *dep.Spec.Replicas
	}

	workload := model.Workload{
		Name:     dep.Name,
		Kind:     "Deployment",
		Replicas: replicas,
		Labels:   dep.Spec.Selector.MatchLabels,
		PVCNames: kube.ExtractPVCNames(dep.Spec.Template.Spec.Volumes),
	}

	ns.Deployments = append(ns.Deployments, workload)
	return nil
}

// parseStatefulSet converts a Kubernetes StatefulSet to a model.Workload
func (p *FixtureParser) parseStatefulSet(doc []byte, ns *model.Namespace) error {
	var ss appsv1.StatefulSet
	if err := yaml.Unmarshal(doc, &ss); err != nil {
		return err
	}

	replicas := int32(1) // Default replica count
	if ss.Spec.Replicas != nil {
		replicas = *ss.Spec.Replicas
	}

	// Extract PVC names including generated names from volumeClaimTemplates
	pvcNames := kube.ExtractAllStatefulSetPVCNames(
		ss.Spec.Template.Spec.Volumes,
		ss.Spec.VolumeClaimTemplates,
		ss.Name,
		replicas,
	)

	workload := model.Workload{
		Name:     ss.Name,
		Kind:     "StatefulSet",
		Replicas: replicas,
		Labels:   ss.Spec.Selector.MatchLabels,
		PVCNames: pvcNames,
	}

	ns.StatefulSets = append(ns.StatefulSets, workload)
	return nil
}

// parseDaemonSet converts a Kubernetes DaemonSet to a model.Workload
func (p *FixtureParser) parseDaemonSet(doc []byte, ns *model.Namespace) error {
	var ds appsv1.DaemonSet
	if err := yaml.Unmarshal(doc, &ds); err != nil {
		return err
	}

	workload := model.Workload{
		Name:     ds.Name,
		Kind:     "DaemonSet",
		Replicas: 0, // DaemonSets don't have a fixed replica count
		Labels:   ds.Spec.Selector.MatchLabels,
		PVCNames: kube.ExtractPVCNames(ds.Spec.Template.Spec.Volumes),
	}

	ns.DaemonSets = append(ns.DaemonSets, workload)
	return nil
}

// parseService converts a Kubernetes Service to a model.Service
func (p *FixtureParser) parseService(doc []byte, ns *model.Namespace) error {
	var svc corev1.Service
	if err := yaml.Unmarshal(doc, &svc); err != nil {
		return err
	}

	service := model.Service{
		Name:     svc.Name,
		Type:     string(svc.Spec.Type),
		Selector: svc.Spec.Selector,
	}

	// Add ports if needed in the future
	for _, port := range svc.Spec.Ports {
		service.Ports = append(service.Ports, model.Port{
			Port:       port.Port,
			TargetPort: port.TargetPort.IntVal,
		})
	}

	ns.Services = append(ns.Services, service)
	return nil
}

// parsePVC converts a Kubernetes PersistentVolumeClaim to a model.PVC
func (p *FixtureParser) parsePVC(doc []byte, ns *model.Namespace) error {
	var pvc corev1.PersistentVolumeClaim
	if err := yaml.Unmarshal(doc, &pvc); err != nil {
		return err
	}

	storageClass := ""
	if pvc.Spec.StorageClassName != nil {
		storageClass = *pvc.Spec.StorageClassName
	}

	capacity := ""
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		capacity = storage.String()
	}

	pvcModel := model.PVC{
		Name:         pvc.Name,
		StorageClass: storageClass,
		Capacity:     capacity,
	}

	ns.PVCs = append(ns.PVCs, pvcModel)
	return nil
}

// parseConfigMap increments the ConfigMap count for the namespace
func (p *FixtureParser) parseConfigMap(doc []byte, ns *model.Namespace) error {
	var cm corev1.ConfigMap
	if err := yaml.Unmarshal(doc, &cm); err != nil {
		return err
	}

	ns.ConfigMaps++
	return nil
}

// parseSecret increments the Secret count for the namespace
func (p *FixtureParser) parseSecret(doc []byte, ns *model.Namespace) error {
	var secret corev1.Secret
	if err := yaml.Unmarshal(doc, &secret); err != nil {
		return err
	}

	ns.Secrets++
	return nil
}
