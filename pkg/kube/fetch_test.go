package kube

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestIsSystemNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		want      bool
	}{
		{name: "default namespace is user-visible", namespace: "default", want: false},
		{name: "kube-system exact name", namespace: "kube-system", want: true},
		{name: "kube-public exact name", namespace: "kube-public", want: true},
		{name: "kube-node-lease exact name", namespace: "kube-node-lease", want: true},
		{name: "local-path-storage exact name", namespace: "local-path-storage", want: true},
		{name: "kube prefix", namespace: "kube-monitoring", want: true},
		{name: "openshift prefix", namespace: "openshift-monitoring", want: true},
		{name: "istio prefix", namespace: "istio-system", want: true},
		{name: "regular namespace", namespace: "team-a", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSystemNamespace(tt.namespace); got != tt.want {
				t.Fatalf("isSystemNamespace(%q) = %t, want %t", tt.namespace, got, tt.want)
			}
		})
	}
}

func TestFilterNamespaceNames(t *testing.T) {
	items := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "openshift-monitoring"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "istio-system"}},
	}

	client := &Client{}

	t.Run("default view excludes only system namespaces", func(t *testing.T) {
		got := client.filterNamespaceNames(items, false)
		want := []string{"default", "team-a"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("filterNamespaceNames(..., false) = %v, want %v", got, want)
		}
	})

	t.Run("all namespaces view keeps every namespace", func(t *testing.T) {
		got := client.filterNamespaceNames(items, true)
		want := []string{"default", "team-a", "kube-system", "openshift-monitoring", "istio-system"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("filterNamespaceNames(..., true) = %v, want %v", got, want)
		}
	})
}

func TestNormalizeNamespaceNames(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "keeps explicit order",
			input: []string{"apps", "default", "observability"},
			want:  []string{"apps", "default", "observability"},
		},
		{
			name:  "trims whitespace and removes empty values",
			input: []string{" apps ", "", " default", "   "},
			want:  []string{"apps", "default"},
		},
		{
			name:  "deduplicates repeated namespaces",
			input: []string{"apps", "default", "apps", "default", "infra"},
			want:  []string{"apps", "default", "infra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeNamespaceNames(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeNamespaceNames(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetNamespacesUsesExplicitNamespaces(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "observability"}},
	)}

	got, err := client.getNamespaces(context.Background(), FetchOptions{
		Namespaces: []string{" apps ", "default", "apps", "observability"},
	})
	if err != nil {
		t.Fatalf("getNamespaces returned error: %v", err)
	}

	want := []string{"apps", "default", "observability"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getNamespaces returned %v, want %v", got, want)
	}
}

func TestGetNamespacesFailsForMissingExplicitNamespace(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps"}},
	)}

	_, err := client.getNamespaces(context.Background(), FetchOptions{
		Namespaces: []string{"default", "missing", "apps"},
	})
	if err == nil {
		t.Fatalf("expected missing explicit namespace to return an error")
	}
	if !strings.Contains(err.Error(), "namespace not found") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing namespace error, got %v", err)
	}
}

func TestGetNamespacesAllowsForbiddenNamespaceValidation(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	)
	clientset.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction, ok := action.(k8stesting.GetAction)
		if !ok || getAction.GetName() != "restricted" {
			return false, nil, nil
		}

		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Resource: "namespaces"},
			"restricted",
			errors.New("forbidden"),
		)
	})

	client := &Client{clientset: clientset}

	got, err := client.getNamespaces(context.Background(), FetchOptions{
		Namespaces: []string{"default", "restricted"},
	})
	if err != nil {
		t.Fatalf("expected forbidden namespace validation to be tolerated, got %v", err)
	}

	want := []string{"default", "restricted"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("getNamespaces returned %v, want %v", got, want)
	}
}

func TestFetchNamespaceAllowsForbiddenIngressList(t *testing.T) {
	clientset := fake.NewSimpleClientset(
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
	)

	clientset.PrependReactor("list", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"},
			"apps",
			errors.New("forbidden"),
		)
	})

	client := &Client{clientset: clientset}

	ns, err := client.fetchNamespace(context.Background(), "apps", FetchOptions{})
	if err != nil {
		t.Fatalf("expected forbidden ingress list to be tolerated, got %v", err)
	}

	if len(ns.Services) != 1 {
		t.Fatalf("expected service fetch to continue, got %d services", len(ns.Services))
	}
	if len(ns.Entrypoints) != 1 {
		t.Fatalf("expected service-derived entrypoint to remain available, got %d", len(ns.Entrypoints))
	}
	if ns.Entrypoints[0].Kind != "NodePort" || ns.Entrypoints[0].Name != "api-service" {
		t.Fatalf("unexpected entrypoints after forbidden ingress list: %#v", ns.Entrypoints)
	}
}
