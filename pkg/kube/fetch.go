package kube

import (
	"context"
	"fmt"
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
		volumeMounts := ExtractVolumeMounts(
			d.Spec.Template.Spec.Containers,
			d.Spec.Template.Spec.Volumes,
		)
		ns.Deployments = append(ns.Deployments, model.Workload{
			Name:         d.Name,
			Kind:         "Deployment",
			Replicas:     *d.Spec.Replicas,
			Labels:       d.Spec.Selector.MatchLabels,
			VolumeMounts: volumeMounts,
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
		// Default to 1 replica if not specified (Kubernetes StatefulSet default)
		replicas := int32(1)
		if ss.Spec.Replicas != nil {
			replicas = *ss.Spec.Replicas
		}

		volumeMounts := ExtractAllStatefulSetVolumeMounts(
			ss.Spec.Template.Spec.Containers,
			ss.Spec.Template.Spec.Volumes,
			ss.Spec.VolumeClaimTemplates,
			ss.Name,
			replicas,
		)
		ns.StatefulSets = append(ns.StatefulSets, model.Workload{
			Name:         ss.Name,
			Kind:         "StatefulSet",
			Replicas:     replicas,
			Labels:       ss.Spec.Selector.MatchLabels,
			VolumeMounts: volumeMounts,
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
		volumeMounts := ExtractVolumeMounts(
			ds.Spec.Template.Spec.Containers,
			ds.Spec.Template.Spec.Volumes,
		)
		ns.DaemonSets = append(ns.DaemonSets, model.Workload{
			Name:         ds.Name,
			Kind:         "DaemonSet",
			Replicas:     ds.Status.DesiredNumberScheduled,
			Labels:       ds.Spec.Selector.MatchLabels,
			VolumeMounts: volumeMounts,
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
		"kube-root-ca.crt",   // Kubernetes cluster CA certificate (injected in all namespaces)
		"istio-ca-root-cert", // Istio service mesh CA certificate
		"linkerd-config",     // Linkerd service mesh configuration
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

// ExtractPVCNames extracts PVC names from a pod's volumes slice.
func ExtractPVCNames(volumes []corev1.Volume) []string {
	var pvcNames []string
	for _, vol := range volumes {
		if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName != "" {
			pvcNames = append(pvcNames, vol.PersistentVolumeClaim.ClaimName)
		}
	}
	return pvcNames
}

// ExtractAllStatefulSetPVCNames extracts all PVC names from a StatefulSet, including
// both regular pod volumes and generated names from volumeClaimTemplates.
// StatefulSet creates PVCs with pattern: <templateName>-<statefulsetName>-<ordinal>
func ExtractAllStatefulSetPVCNames(volumes []corev1.Volume, templates []corev1.PersistentVolumeClaim, ssName string, replicas int32) []string {
	// Start with regular pod volumes
	pvcNames := ExtractPVCNames(volumes)

	// Add generated names from volumeClaimTemplates
	for _, vct := range templates {
		for i := range replicas {
			pvcName := fmt.Sprintf("%s-%s-%d", vct.Name, ssName, i)
			pvcNames = append(pvcNames, pvcName)
		}
	}

	return pvcNames
}

// ExtractVolumeMounts extracts volume mounts from containers and correlates
// them with PVC volumes to build VolumeMount objects with mount metadata.
func ExtractVolumeMounts(
	containers []corev1.Container,
	volumes []corev1.Volume,
) []model.VolumeMount {
	// Build map: volume name → PVC claim name
	volumeToPVC := make(map[string]string)
	for _, vol := range volumes {
		if vol.PersistentVolumeClaim != nil {
			volumeToPVC[vol.Name] = vol.PersistentVolumeClaim.ClaimName
		}
	}

	// Extract mounts from all containers
	var mounts []model.VolumeMount
	for _, container := range containers {
		for _, vm := range container.VolumeMounts {
			// Only include mounts that reference PVC volumes
			if pvcName, ok := volumeToPVC[vm.Name]; ok {
				mounts = append(mounts, model.VolumeMount{
					PVCName:   pvcName,
					MountPath: vm.MountPath,
					ReadOnly:  vm.ReadOnly,
				})
			}
		}
	}

	return mounts
}

// ExtractAllStatefulSetVolumeMounts handles both regular volumes and
// volumeClaimTemplates for StatefulSets, generating VolumeMount entries
// for each replica's PVC.
func ExtractAllStatefulSetVolumeMounts(
	containers []corev1.Container,
	volumes []corev1.Volume,
	templates []corev1.PersistentVolumeClaim,
	ssName string,
	replicas int32,
) []model.VolumeMount {
	// Start with regular volume mounts
	mounts := ExtractVolumeMounts(containers, volumes)

	// Build map: template name → mount metadata from containers
	templateMounts := make(map[string][]model.VolumeMount)
	for _, container := range containers {
		for _, vm := range container.VolumeMounts {
			// Check if this mount references a volumeClaimTemplate
			for _, vct := range templates {
				if vm.Name == vct.Name {
					templateMounts[vct.Name] = append(
						templateMounts[vct.Name],
						model.VolumeMount{
							MountPath: vm.MountPath,
							ReadOnly:  vm.ReadOnly,
						},
					)
				}
			}
		}
	}

	// Generate mounts for each replica's generated PVC
	// Pattern: <templateName>-<statefulsetName>-<ordinal>
	for _, vct := range templates {
		mountTemplates := templateMounts[vct.Name]
		for i := range replicas {
			pvcName := fmt.Sprintf("%s-%s-%d", vct.Name, ssName, i)
			for _, mt := range mountTemplates {
				mounts = append(mounts, model.VolumeMount{
					PVCName:   pvcName,
					MountPath: mt.MountPath,
					ReadOnly:  mt.ReadOnly,
				})
			}
		}
	}

	return mounts
}
