package kube_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/vieitesss/k8s-d2/internal/validation"
	"github.com/vieitesss/k8s-d2/pkg/kube"
	"github.com/vieitesss/k8s-d2/pkg/model"
	"github.com/vieitesss/k8s-d2/pkg/render"
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

func TestNormalizeDaemonSetUsesDesiredNumberScheduled(t *testing.T) {
	daemonSet := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "node-agent"},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-agent"}},
		},
		Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 3},
	}

	got := kube.NormalizeDaemonSet(daemonSet)
	if got.Replicas != 3 {
		t.Fatalf("daemonset replicas = %d, want 3", got.Replicas)
	}
}

func TestFetchTopologyAndFixtureParserProduceEquivalentRenderedTopology(t *testing.T) {
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
	fixtureDaemonSet := daemonSet
	fixtureDaemonSet.Status = appsv1.DaemonSetStatus{}

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
		&fixtureDaemonSet,
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

	if liveOutput, fixtureOutput := renderCluster(t, liveCluster), renderCluster(t, fixtureCluster); liveOutput != fixtureOutput {
		t.Fatalf("live and fixture rendered topology diverged\nlive:\n%s\nfixture:\n%s", liveOutput, fixtureOutput)
	}

	if got := liveCluster.Namespaces[0].DaemonSets[0].Replicas; got != 3 {
		t.Fatalf("live daemonset replicas = %d, want 3", got)
	}
	if got := fixtureCluster.Namespaces[0].DaemonSets[0].Replicas; got != 0 {
		t.Fatalf("fixture daemonset replicas = %d, want 0", got)
	}

	for _, tc := range []struct {
		name string
		ns   model.Namespace
	}{
		{name: "live", ns: liveCluster.Namespaces[0]},
		{name: "fixture", ns: fixtureCluster.Namespaces[0]},
	} {
		if got := tc.ns.Deployments[0].Replicas; got != 1 {
			t.Fatalf("%s deployment replicas = %d, want 1", tc.name, got)
		}
		if got := findPVC(t, tc.ns.PVCs, "logs-volume").Capacity; got != "1Gi" {
			t.Fatalf("%s logs-volume capacity = %q, want %q", tc.name, got, "1Gi")
		}
		if tc.ns.ConfigMaps != 1 {
			t.Fatalf("%s configmap count = %d, want 1", tc.name, tc.ns.ConfigMaps)
		}
		if tc.ns.Secrets != 1 {
			t.Fatalf("%s secret count = %d, want 1", tc.name, tc.ns.Secrets)
		}
		for _, name := range []string{"data-database-0", "data-database-1"} {
			pvc := findPVC(t, tc.ns.PVCs, name)
			if pvc.Capacity != "2Gi" {
				t.Fatalf("%s generated pvc %s capacity = %q, want %q", tc.name, name, pvc.Capacity, "2Gi")
			}
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

func renderCluster(t *testing.T, cluster *model.Cluster) string {
	t.Helper()

	var buf bytes.Buffer
	renderer := render.NewD2Renderer(&buf, 0)
	if err := renderer.Render(cluster); err != nil {
		t.Fatalf("render cluster: %v", err)
	}

	return buf.String()
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
