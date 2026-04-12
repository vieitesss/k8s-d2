package kube

import (
	"fmt"

	"github.com/vieitesss/k8s-d2/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// NormalizeDeployment converts a Kubernetes Deployment into the shared workload model.
func NormalizeDeployment(dep appsv1.Deployment) model.Workload {
	return model.Workload{
		Name:         dep.Name,
		Kind:         "Deployment",
		Replicas:     replicasOrDefault(dep.Spec.Replicas),
		Labels:       dep.Spec.Selector.MatchLabels,
		VolumeMounts: ExtractVolumeMounts(dep.Spec.Template.Spec.Containers, dep.Spec.Template.Spec.Volumes),
	}
}

// NormalizeStatefulSet converts a Kubernetes StatefulSet into the shared workload model.
func NormalizeStatefulSet(ss appsv1.StatefulSet) model.Workload {
	replicas := replicasOrDefault(ss.Spec.Replicas)

	return model.Workload{
		Name:         ss.Name,
		Kind:         "StatefulSet",
		Replicas:     replicas,
		Labels:       ss.Spec.Selector.MatchLabels,
		VolumeMounts: ExtractAllStatefulSetVolumeMounts(ss.Spec.Template.Spec.Containers, ss.Spec.Template.Spec.Volumes, ss.Spec.VolumeClaimTemplates, ss.Name, replicas),
	}
}

// NormalizeDaemonSet converts a Kubernetes DaemonSet into the shared workload model.
func NormalizeDaemonSet(ds appsv1.DaemonSet) model.Workload {
	return model.Workload{
		Name: ds.Name,
		Kind: "DaemonSet",
		// DaemonSet scheduling is runtime status data that fixture manifests usually omit.
		Replicas:     0,
		Labels:       ds.Spec.Selector.MatchLabels,
		VolumeMounts: ExtractVolumeMounts(ds.Spec.Template.Spec.Containers, ds.Spec.Template.Spec.Volumes),
	}
}

// NormalizeService converts a Kubernetes Service into the shared service model.
func NormalizeService(svc corev1.Service) model.Service {
	ports := make([]model.Port, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		ports = append(ports, model.Port{
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: ServiceTargetPort(port),
			NodePort:   port.NodePort,
		})
	}

	return model.Service{
		Name:     svc.Name,
		Type:     string(svc.Spec.Type),
		Selector: svc.Spec.Selector,
		Ports:    ports,
	}
}

// NormalizePVC converts a Kubernetes PersistentVolumeClaim into the shared PVC model.
func NormalizePVC(pvc corev1.PersistentVolumeClaim) model.PVC {
	storageClass := ""
	if pvc.Spec.StorageClassName != nil {
		storageClass = *pvc.Spec.StorageClassName
	}

	return model.PVC{
		Name:         pvc.Name,
		StorageClass: storageClass,
		Capacity:     pvcCapacity(pvc),
		BoundPod:     "", // TODO: determine which pod uses this PVC
	}
}

// NormalizeStatefulSetTemplatePVCs synthesizes the PVCs created from StatefulSet
// volumeClaimTemplates so fixture-based validation can match live cluster output.
func NormalizeStatefulSetTemplatePVCs(ss appsv1.StatefulSet) []model.PVC {
	replicas := replicasOrDefault(ss.Spec.Replicas)
	pvcs := make([]model.PVC, 0, len(ss.Spec.VolumeClaimTemplates)*int(replicas))

	for _, template := range ss.Spec.VolumeClaimTemplates {
		storageClass := ""
		if template.Spec.StorageClassName != nil {
			storageClass = *template.Spec.StorageClassName
		}

		capacity := requestedStorageCapacity(template.Spec.Resources.Requests)
		for i := range replicas {
			pvcs = append(pvcs, model.PVC{
				Name:         fmt.Sprintf("%s-%s-%d", template.Name, ss.Name, i),
				StorageClass: storageClass,
				Capacity:     capacity,
			})
		}
	}

	return pvcs
}

// CountUserConfigMaps returns the number of non-system ConfigMaps.
func CountUserConfigMaps(configMaps []corev1.ConfigMap) int {
	count := 0
	for _, cm := range configMaps {
		if !IsSystemConfigMap(cm.Name) {
			count++
		}
	}
	return count
}

// CountUserSecrets returns the number of non-system Secrets.
func CountUserSecrets(secrets []corev1.Secret) int {
	count := 0
	for _, secret := range secrets {
		if !IsSystemSecret(secret.Name, secret.Type) {
			count++
		}
	}
	return count
}

// IsSystemConfigMap reports whether a ConfigMap should be excluded from topology counts.
func IsSystemConfigMap(name string) bool {
	return isSystemConfigMap(name)
}

// IsSystemSecret reports whether a Secret should be excluded from topology counts.
func IsSystemSecret(name string, secretType corev1.SecretType) bool {
	return isSystemSecret(name, secretType)
}

func replicasOrDefault(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

func pvcCapacity(pvc corev1.PersistentVolumeClaim) string {
	if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return requestedStorageCapacity(pvc.Spec.Resources.Requests)
}

func requestedStorageCapacity(requests corev1.ResourceList) string {
	if storage, ok := requests[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return ""
}
