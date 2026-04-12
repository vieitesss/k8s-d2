package kube_test

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestFetchTopologyAndFixtureParserProduceEquivalentTopology(t *testing.T) {
	const namespace = "apps"

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "logs",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "logs-volume"},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "api",
						Image: "registry.k8s.io/pause:3.10",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "logs",
							MountPath: "/var/log/api",
						}},
					}},
				},
			},
		},
	}

	statefulSet := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database",
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "database",
			Replicas:    int32Ptr(2),
			Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"app": "database"}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "postgres",
						Image: "registry.k8s.io/pause:3.10",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "data",
							MountPath: "/var/lib/postgresql",
						}},
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "data"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: stringPtr("standard"),
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
					},
				},
			}},
		},
	}

	daemonSet := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-agent",
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-agent"}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "agent",
						Image: "registry.k8s.io/pause:3.10",
					}},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 3},
	}

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-service",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{"app": "api"},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("web"),
				NodePort:   30080,
			}},
		},
	}

	ingress := networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public-edge",
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("nginx"),
			Rules: []networkingv1.IngressRule{{
				Host: "apps.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{Name: "api-service"},
							},
						}},
					},
				},
			}},
		},
	}

	userConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-config",
			Namespace: namespace,
		},
	}
	userSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-credentials",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
	}
	systemConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-root-ca.crt",
			Namespace: namespace,
		},
	}
	systemSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-token-abcde",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	explicitPVC := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logs-volume",
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("fast"),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("500Mi")},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
		},
	}

	generatedPVC0 := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-database-0",
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("standard"),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
		},
	}
	generatedPVC1 := generatedPVC0
	generatedPVC1.ObjectMeta = metav1.ObjectMeta{Name: "data-database-1", Namespace: namespace}

	client := kube.NewClientFromClientset(fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
		&deployment,
		&statefulSet,
		&daemonSet,
		&service,
		&ingress,
		&userConfigMap,
		&userSecret,
		&systemConfigMap,
		&systemSecret,
		&explicitPVC,
		&generatedPVC0,
		&generatedPVC1,
	))

	liveCluster, err := client.FetchTopology(context.Background(), kube.FetchOptions{
		Namespaces:     []string{namespace},
		IncludeStorage: true,
	})
	if err != nil {
		t.Fatalf("FetchTopology returned error: %v", err)
	}

	fixtures := marshalFixtures(t,
		&corev1.Namespace{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}, ObjectMeta: metav1.ObjectMeta{Name: namespace}},
		&deployment,
		&statefulSet,
		&daemonSet,
		&service,
		&ingress,
		&userConfigMap,
		&userSecret,
		&systemConfigMap,
		&systemSecret,
		&explicitPVC,
	)

	fixtureCluster, err := validation.NewFixtureParser(namespace, true).ParseFixtures(fixtures)
	if err != nil {
		t.Fatalf("ParseFixtures returned error: %v", err)
	}

	sortClusterForComparison(liveCluster)
	sortClusterForComparison(fixtureCluster)

	if !reflect.DeepEqual(liveCluster, fixtureCluster) {
		t.Fatalf("live and fixture topology diverged\nlive: %#v\nfixture: %#v", liveCluster, fixtureCluster)
	}

	ns := liveCluster.Namespaces[0]
	if got := ns.Deployments[0].Replicas; got != 1 {
		t.Fatalf("deployment replicas = %d, want 1", got)
	}
	if got := ns.DaemonSets[0].Replicas; got != 3 {
		t.Fatalf("daemonset replicas = %d, want 3", got)
	}
	if got := findPVC(t, ns.PVCs, "logs-volume").Capacity; got != "1Gi" {
		t.Fatalf("logs-volume capacity = %q, want %q", got, "1Gi")
	}
	if ns.ConfigMaps != 1 {
		t.Fatalf("configmap count = %d, want 1", ns.ConfigMaps)
	}
	if ns.Secrets != 1 {
		t.Fatalf("secret count = %d, want 1", ns.Secrets)
	}
	for _, name := range []string{"data-database-0", "data-database-1"} {
		pvc := findPVC(t, ns.PVCs, name)
		if pvc.Capacity != "2Gi" {
			t.Fatalf("generated pvc %s capacity = %q, want %q", name, pvc.Capacity, "2Gi")
		}
	}
}

func marshalFixtures(t *testing.T, objects ...runtime.Object) [][]byte {
	t.Helper()

	fixtures := make([][]byte, 0, len(objects))
	for _, object := range objects {
		data, err := yaml.Marshal(object)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		fixtures = append(fixtures, data)
	}

	return fixtures
}

func sortClusterForComparison(cluster *model.Cluster) {
	sort.Slice(cluster.Namespaces, func(i, j int) bool {
		return cluster.Namespaces[i].Name < cluster.Namespaces[j].Name
	})

	for i := range cluster.Namespaces {
		ns := &cluster.Namespaces[i]
		sort.Slice(ns.Deployments, func(i, j int) bool { return ns.Deployments[i].Name < ns.Deployments[j].Name })
		sort.Slice(ns.StatefulSets, func(i, j int) bool { return ns.StatefulSets[i].Name < ns.StatefulSets[j].Name })
		sort.Slice(ns.DaemonSets, func(i, j int) bool { return ns.DaemonSets[i].Name < ns.DaemonSets[j].Name })
		sort.Slice(ns.Entrypoints, func(i, j int) bool {
			if ns.Entrypoints[i].Kind != ns.Entrypoints[j].Kind {
				return ns.Entrypoints[i].Kind < ns.Entrypoints[j].Kind
			}
			return ns.Entrypoints[i].Name < ns.Entrypoints[j].Name
		})
		sort.Slice(ns.Services, func(i, j int) bool { return ns.Services[i].Name < ns.Services[j].Name })
		sort.Slice(ns.PVCs, func(i, j int) bool { return ns.PVCs[i].Name < ns.PVCs[j].Name })
	}
}

func findPVC(t *testing.T, pvcs []model.PVC, name string) model.PVC {
	t.Helper()

	for _, pvc := range pvcs {
		if pvc.Name == name {
			return pvc
		}
	}

	t.Fatalf("missing pvc %q", name)
	return model.PVC{}
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
