package kube

import (
	"context"
	"slices"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FetchOptions struct {
	Namespace      string
	AllNamespaces  bool
	IncludeStorage bool
}

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

	var names []string
	for _, ns := range list.Items {
		if !opts.AllNamespaces && isSystemNamespace(ns.Name) {
			continue
		}
		names = append(names, ns.Name)
	}
	return names, nil
}

func (c *Client) fetchNamespace(ctx context.Context, nsName string, opts FetchOptions) (*model.Namespace, error) {
	ns := &model.Namespace{Name: nsName}

	deps, err := c.clientset.AppsV1().Deployments(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deps.Items {
		ns.Deployments = append(ns.Deployments, model.Workload{
			Name:     d.Name,
			Kind:     "Deployment",
			Replicas: *d.Spec.Replicas,
			Labels:   d.Spec.Selector.MatchLabels,
		})
	}

	ssets, err := c.clientset.AppsV1().StatefulSets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ss := range ssets.Items {
		ns.StatefulSets = append(ns.StatefulSets, model.Workload{
			Name:     ss.Name,
			Kind:     "StatefulSet",
			Replicas: *ss.Spec.Replicas,
			Labels:   ss.Spec.Selector.MatchLabels,
		})
	}

	dsets, err := c.clientset.AppsV1().DaemonSets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ds := range dsets.Items {
		ns.DaemonSets = append(ns.DaemonSets, model.Workload{
			Name:     ds.Name,
			Kind:     "DaemonSet",
			Replicas: ds.Status.DesiredNumberScheduled,
			Labels:   ds.Spec.Selector.MatchLabels,
		})
	}

	svcs, err := c.clientset.CoreV1().Services(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
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

	cms, err := c.clientset.CoreV1().ConfigMaps(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ns.ConfigMaps = len(cms.Items)

	secrets, err := c.clientset.CoreV1().Secrets(nsName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ns.Secrets = len(secrets.Items)

	if opts.IncludeStorage {
		pvcs, err := c.clientset.CoreV1().PersistentVolumeClaims(nsName).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
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
	}

	return ns, nil
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
