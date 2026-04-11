package kube

import (
	"context"
	"reflect"
	"testing"

	"github.com/vieitesss/k8s-d2/pkg/model"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEntrypointForService(t *testing.T) {
	tests := []struct {
		name    string
		service model.Service
		want    model.Entrypoint
		ok      bool
	}{
		{
			name: "clusterip is not external entrypoint",
			service: model.Service{
				Name: "web",
				Type: "ClusterIP",
			},
			ok: false,
		},
		{
			name: "nodeport becomes entrypoint",
			service: model.Service{
				Name:  "api",
				Type:  "NodePort",
				Ports: []model.Port{{Port: 8080, NodePort: 30080}},
			},
			want: model.Entrypoint{
				Name:     "api",
				Kind:     "NodePort",
				Services: []string{"api"},
				Ports:    []model.Port{{Port: 8080, NodePort: 30080}},
			},
			ok: true,
		},
		{
			name: "loadbalancer becomes entrypoint",
			service: model.Service{
				Name:  "public",
				Type:  "LoadBalancer",
				Ports: []model.Port{{Port: 443}},
			},
			want: model.Entrypoint{
				Name:     "public",
				Kind:     "LoadBalancer",
				Services: []string{"public"},
				Ports:    []model.Port{{Port: 443}},
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := EntrypointForService(tt.service)
			if ok != tt.ok {
				t.Fatalf("EntrypointForService() ok = %t, want %t", ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("EntrypointForService() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEntrypointForIngress(t *testing.T) {
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "public-edge",
			Annotations: map[string]string{
				ingressClassAnnotation: "nginx",
			},
		},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "api-service",
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "b.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{Name: "web-service"},
								},
							}},
						},
					},
				},
				{
					Host: "a.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{Name: "api-service"},
								},
							}},
						},
					},
				},
			},
		},
	}

	got := EntrypointForIngress(ingress)
	want := model.Entrypoint{
		Name:     "public-edge",
		Kind:     "Ingress",
		Class:    "nginx",
		Hosts:    []string{"a.example.com", "b.example.com"},
		Services: []string{"api-service", "web-service"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EntrypointForIngress() = %#v, want %#v", got, want)
	}
}

func TestFetchNamespaceAddsEntrypoints(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-service", Namespace: "apps"},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{{
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					NodePort:   30080,
				}},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web-service", Namespace: "apps"},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				}},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "public-edge", Namespace: "apps"},
			Spec: networkingv1.IngressSpec{
				IngressClassName: stringPtr("nginx"),
				Rules: []networkingv1.IngressRule{{
					Host: "apps.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{Name: "web-service"},
								},
							}},
						},
					},
				}},
			},
		},
	)}

	ns, err := client.fetchNamespace(context.Background(), "apps", FetchOptions{})
	if err != nil {
		t.Fatalf("fetchNamespace returned error: %v", err)
	}

	if len(ns.Entrypoints) != 2 {
		t.Fatalf("expected 2 entrypoints, got %d: %#v", len(ns.Entrypoints), ns.Entrypoints)
	}

	want := map[string]model.Entrypoint{
		"NodePort/api-service": {
			Name:     "api-service",
			Kind:     "NodePort",
			Services: []string{"api-service"},
			Ports:    []model.Port{{Port: 8080, TargetPort: "8080", NodePort: 30080}},
		},
		"Ingress/public-edge": {
			Name:     "public-edge",
			Kind:     "Ingress",
			Class:    "nginx",
			Hosts:    []string{"apps.example.com"},
			Services: []string{"web-service"},
		},
	}

	for _, entrypoint := range ns.Entrypoints {
		key := entrypoint.Kind + "/" + entrypoint.Name
		expected, ok := want[key]
		if !ok {
			t.Fatalf("unexpected entrypoint returned: %#v", entrypoint)
		}
		if !reflect.DeepEqual(entrypoint, expected) {
			t.Fatalf("entrypoint %q = %#v, want %#v", key, entrypoint, expected)
		}
	}
}

func stringPtr(value string) *string {
	return &value
}
