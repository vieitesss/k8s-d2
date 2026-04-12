package validation

import (
	"bytes"

	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/yaml"
)

// FixtureParser parses Kubernetes YAML fixtures into internal model types
type FixtureParser struct {
	namespace      string
	includeStorage bool
}

// NewFixtureParser creates a new FixtureParser for the given namespace.
// When includeStorage is true, it also synthesizes StatefulSet-generated PVCs
// so fixture-based expectations match the live cluster fetch path.
func NewFixtureParser(namespace string, includeStorage bool) *FixtureParser {
	return &FixtureParser{namespace: namespace, includeStorage: includeStorage}
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
	case "Ingress":
		return p.parseIngress(doc, ns)
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

	ns.Deployments = append(ns.Deployments, kube.NormalizeDeployment(dep))
	return nil
}

// parseStatefulSet converts a Kubernetes StatefulSet to a model.Workload
func (p *FixtureParser) parseStatefulSet(doc []byte, ns *model.Namespace) error {
	var ss appsv1.StatefulSet
	if err := yaml.Unmarshal(doc, &ss); err != nil {
		return err
	}

	ns.StatefulSets = append(ns.StatefulSets, kube.NormalizeStatefulSet(ss))
	if p.includeStorage {
		ns.PVCs = append(ns.PVCs, kube.NormalizeStatefulSetTemplatePVCs(ss)...)
	}

	return nil
}

// parseDaemonSet converts a Kubernetes DaemonSet to a model.Workload
func (p *FixtureParser) parseDaemonSet(doc []byte, ns *model.Namespace) error {
	var ds appsv1.DaemonSet
	if err := yaml.Unmarshal(doc, &ds); err != nil {
		return err
	}

	ns.DaemonSets = append(ns.DaemonSets, kube.NormalizeDaemonSet(ds))
	return nil
}

// parseService converts a Kubernetes Service to a model.Service
func (p *FixtureParser) parseService(doc []byte, ns *model.Namespace) error {
	var svc corev1.Service
	if err := yaml.Unmarshal(doc, &svc); err != nil {
		return err
	}

	service := kube.NormalizeService(svc)

	ns.Services = append(ns.Services, service)
	if entrypoint, ok := kube.EntrypointForService(service); ok {
		ns.Entrypoints = append(ns.Entrypoints, entrypoint)
	}
	return nil
}

// parseIngress converts a Kubernetes Ingress to a model.Entrypoint.
func (p *FixtureParser) parseIngress(doc []byte, ns *model.Namespace) error {
	var ing networkingv1.Ingress
	if err := yaml.Unmarshal(doc, &ing); err != nil {
		return err
	}

	ns.Entrypoints = append(ns.Entrypoints, kube.EntrypointForIngress(ing))
	return nil
}

// parsePVC converts a Kubernetes PersistentVolumeClaim to a model.PVC
func (p *FixtureParser) parsePVC(doc []byte, ns *model.Namespace) error {
	var pvc corev1.PersistentVolumeClaim
	if err := yaml.Unmarshal(doc, &pvc); err != nil {
		return err
	}

	ns.PVCs = append(ns.PVCs, kube.NormalizePVC(pvc))
	return nil
}

// parseConfigMap increments the ConfigMap count for the namespace
func (p *FixtureParser) parseConfigMap(doc []byte, ns *model.Namespace) error {
	var cm corev1.ConfigMap
	if err := yaml.Unmarshal(doc, &cm); err != nil {
		return err
	}

	if !kube.IsSystemConfigMap(cm.Name) {
		ns.ConfigMaps++
	}
	return nil
}

// parseSecret increments the Secret count for the namespace
func (p *FixtureParser) parseSecret(doc []byte, ns *model.Namespace) error {
	var secret corev1.Secret
	if err := yaml.Unmarshal(doc, &secret); err != nil {
		return err
	}

	if !kube.IsSystemSecret(secret.Name, secret.Type) {
		ns.Secrets++
	}
	return nil
}
