package kube

import (
	"sort"
	"strings"

	"github.com/vieitesss/k8s-d2/pkg/model"
	networkingv1 "k8s.io/api/networking/v1"
)

const ingressClassAnnotation = "kubernetes.io/ingress.class"

// EntrypointForService derives an external entrypoint for Services that are
// directly reachable from outside the cluster.
func EntrypointForService(svc model.Service) (model.Entrypoint, bool) {
	switch svc.Type {
	case "NodePort", "LoadBalancer":
		return model.Entrypoint{
			Name:     svc.Name,
			Kind:     svc.Type,
			Services: []string{svc.Name},
			Ports:    append([]model.Port(nil), svc.Ports...),
		}, true
	default:
		return model.Entrypoint{}, false
	}
}

// EntrypointForIngress converts a Kubernetes Ingress resource into the shared
// entrypoint model used by the renderer and validator.
func EntrypointForIngress(ing networkingv1.Ingress) model.Entrypoint {
	hosts := make([]string, 0, len(ing.Spec.Rules))
	services := []string{}

	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		serviceName := strings.TrimSpace(ing.Spec.DefaultBackend.Service.Name)
		if serviceName != "" {
			services = append(services, serviceName)
		}
	}

	for _, rule := range ing.Spec.Rules {
		if host := strings.TrimSpace(rule.Host); host != "" {
			hosts = append(hosts, host)
		}

		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}

			serviceName := strings.TrimSpace(path.Backend.Service.Name)
			if serviceName == "" {
				continue
			}

			services = append(services, serviceName)
		}
	}

	class := ""
	if ing.Spec.IngressClassName != nil {
		class = strings.TrimSpace(*ing.Spec.IngressClassName)
	}
	if class == "" {
		class = strings.TrimSpace(ing.Annotations[ingressClassAnnotation])
	}

	return model.Entrypoint{
		Name:     ing.Name,
		Kind:     "Ingress",
		Class:    class,
		Hosts:    uniqueSortedStrings(hosts),
		Services: uniqueSortedStrings(services),
	}
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	sort.Strings(unique)
	return unique
}
