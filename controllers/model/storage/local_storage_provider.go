package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
)

type LocalStorageProvider struct {
}

// add the hostpath volume and mountPath in each container
func (a *LocalStorageProvider) AddModelVolumeToPodSpec(mv *modelv1alpha1.Storage, pod *corev1.PodTemplateSpec) {

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: "modelvolume",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: mv.LocalStorage.Path,
				},
			}})

	for i, c := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name: "modelvolume", MountPath: mv.LocalStorage.MountPath,
			})
	}
}

func (ls *LocalStorageProvider) CreatePersistentVolume(storage *modelv1alpha1.Storage, pvName string) *corev1.PersistentVolume {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},

		Spec: corev1.PersistentVolumeSpec{
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: storage.LocalStorage.Path,
				},
			},
			Capacity: corev1.ResourceList{
				// the 500Mi capacity is not enforced for local path volume.
				// This is specified because api-server validation checks a capacity value to be present.
				corev1.ResourceStorage: resource.MustParse("500Mi"),
			},
			StorageClassName: "",
		},
	}
	pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
		Required: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{storage.LocalStorage.NodeName},
						},
					},
				},
			},
		},
	}
	return pv
}

func (a *LocalStorageProvider) GetModelMountPath(mv *modelv1alpha1.Storage) string {
	return mv.LocalStorage.MountPath
}

func NewLocalStorageProvider() StorageProvider {
	return &LocalStorageProvider{}
}
