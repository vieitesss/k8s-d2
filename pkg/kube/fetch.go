package kube

import (
	"context"
	"slices"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FetchOptions struct {
	Namespace      string
	AllNamespaces  bool
	IncludeStorage bool
}

// namespaceFetcher is a function that fetches a specific resource type into a namespace.
type namespaceFetcher func(context.Context, string, *model.Namespace) error

func (c *Client) FetchTopology(ctx context.Context, opts FetchOptions) (*model.Cluster, error) {
	cluster := &model.Cluster{Name: "cluster"}

	namespaces, err := c.getNamespaces(ctx, opts)
	if err != nil {
		return nil, err
	}

	for _, nsName := range namespaces {
		ns, err := c.fetchNamespace(ctx, nsName, opts)
		if err != nil {
			return nil, err
		}
		cluster.Namespaces = append(cluster.Namespaces, *ns)
	}

	return cluster, nil
}

func (c *Client) getNamespaces(ctx context.Context, opts FetchOptions) ([]string, error) {
	if opts.Namespace != "" {
		return []string{opts.Namespace}, nil
	}

	list, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return c.filterNamespaceNames(list.Items, opts.AllNamespaces), nil
}

func (c *Client) filterNamespaceNames(items []corev1.Namespace, includeSystemNamespaces bool) []string {
	var names []string
	for _, ns := range items {
		if !includeSystemNamespaces && isSystemNamespace(ns.Name) {
			continue
		}
		names = append(names, ns.Name)
	}
	return names
}

func (c *Client) fetchNamespace(ctx context.Context, nsName string, opts FetchOptions) (*model.Namespace, error) {
	ns := &model.Namespace{Name: nsName}

	fetchers := []namespaceFetcher{
		c.fetchDeployments,
		c.fetchStatefulSets,
		c.fetchDaemonSets,
		c.fetchServices,
		c.fetchConfigMapsAndSecrets,
	}

	if opts.IncludeStorage {
		fetchers = append(fetchers, c.fetchPVCs)
	}

	for _, fetch := range fetchers {
		if err := fetch(ctx, nsName, ns); err != nil {
			return nil, err
		}
	}

	return ns, nil
}

func (c *Client) fetchDeployments(ctx context.Context, nsName string, ns *model.Namespace) error {
	deps, err := c.clientset.AppsV1().Deployments(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, d := range deps.Items {
		ns.Deployments = append(ns.Deployments, model.Workload{
			Name:     d.Name,
			Kind:     "Deployment",
			Replicas: *d.Spec.Replicas,
			Labels:   d.Spec.Selector.MatchLabels,
		})
	}
	return nil
}

func (c *Client) fetchStatefulSets(ctx context.Context, nsName string, ns *model.Namespace) error {
	ssets, err := c.clientset.AppsV1().StatefulSets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ss := range ssets.Items {
		ns.StatefulSets = append(ns.StatefulSets, model.Workload{
			Name:     ss.Name,
			Kind:     "StatefulSet",
			Replicas: *ss.Spec.Replicas,
			Labels:   ss.Spec.Selector.MatchLabels,
		})
	}
	return nil
}

func (c *Client) fetchDaemonSets(ctx context.Context, nsName string, ns *model.Namespace) error {
	dsets, err := c.clientset.AppsV1().DaemonSets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ds := range dsets.Items {
		ns.DaemonSets = append(ns.DaemonSets, model.Workload{
			Name:     ds.Name,
			Kind:     "DaemonSet",
			Replicas: ds.Status.DesiredNumberScheduled,
			Labels:   ds.Spec.Selector.MatchLabels,
		})
	}
	return nil
}

func (c *Client) fetchServices(ctx context.Context, nsName string, ns *model.Namespace) error {
	svcs, err := c.clientset.CoreV1().Services(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, svc := range svcs.Items {
		ports := []model.Port{}
		for _, p := range svc.Spec.Ports {
			ports = append(ports, model.Port{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.IntVal,
			})
		}
		ns.Services = append(ns.Services, model.Service{
			Name:     svc.Name,
			Type:     string(svc.Spec.Type),
			Selector: svc.Spec.Selector,
			Ports:    ports,
		})
	}
	return nil
}

func (c *Client) fetchConfigMapsAndSecrets(ctx context.Context, nsName string, ns *model.Namespace) error {
	cms, err := c.clientset.CoreV1().ConfigMaps(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	// Filter out system-managed ConfigMaps
	userConfigMaps := 0
	for _, cm := range cms.Items {
		if !isSystemConfigMap(cm.Name) {
			userConfigMaps++
		}
	}
	ns.ConfigMaps = userConfigMaps

	secrets, err := c.clientset.CoreV1().Secrets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	// Filter out system-managed Secrets (service account tokens)
	userSecrets := 0
	for _, secret := range secrets.Items {
		if !isSystemSecret(secret.Name, secret.Type) {
			userSecrets++
		}
	}
	ns.Secrets = userSecrets

	return nil
}

func (c *Client) fetchPVCs(ctx context.Context, nsName string, ns *model.Namespace) error {
	pvcs, err := c.clientset.CoreV1().PersistentVolumeClaims(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}
		capacity := ""
		if storage, ok := pvc.Status.Capacity["storage"]; ok {
			capacity = storage.String()
		}
		ns.PVCs = append(ns.PVCs, model.PVC{
			Name:         pvc.Name,
			StorageClass: storageClass,
			Capacity:     capacity,
			BoundPod:     "", // TODO: determine which pod uses this PVC
		})
	}
	return nil
}

func isSystemNamespace(name string) bool {
	systemPrefixes := []string{"kube-", "openshift-", "istio-"}
	systemNames := []string{"default", "kube-system", "kube-public", "kube-node-lease"}

	if slices.Contains(systemNames, name) {
		return true
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isSystemConfigMap(name string) bool {
	// Known system-managed ConfigMaps
	systemConfigMaps := []string{
		"kube-root-ca.crt",      // Kubernetes cluster CA certificate (injected in all namespaces)
		"istio-ca-root-cert",    // Istio service mesh CA certificate
		"linkerd-config",        // Linkerd service mesh configuration
	}

	if slices.Contains(systemConfigMaps, name) {
		return true
	}

	// Filter ConfigMaps with system prefixes
	systemPrefixes := []string{"kube-", "openshift-"}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

func isSystemSecret(name string, secretType corev1.SecretType) bool {
	// Filter out service account token secrets
	if secretType == corev1.SecretTypeServiceAccountToken {
		return true
	}

	// Filter out other system secrets by prefix
	systemPrefixes := []string{"default-token-", "sh.helm."}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}
