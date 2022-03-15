package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
)

type NFSProvider struct {
}

func (a *NFSProvider) AddModelVolumeToPodSpec(mv *modelv1alpha1.Storage, pod *corev1.PodTemplateSpec) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: "modelvolume",
			VolumeSource: corev1.VolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Path:   mv.NFS.Path,
					Server: mv.NFS.Server,
				},
			}})

	for _, c := range pod.Spec.Containers {
		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name: "modelvolume", MountPath: mv.LocalStorage.MountPath,
			})
	}
}

func (a *NFSProvider) CreatePersistentVolume(storage *modelv1alpha1.Storage, pvName string) *corev1.PersistentVolume {

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				// the 1Gi capacity is not enforced.
				// This is specified because api-server validation checks a capacity value to be present.
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: storage.NFS.Server,
					Path:   storage.NFS.Path,
				},
			},
			StorageClassName: "",
		},
	}
	return pv
}

func (a *NFSProvider) GetModelMountPath(mv *modelv1alpha1.Storage) string {
	return mv.LocalStorage.MountPath
}

func NewNFSProvider() StorageProvider {
	return &NFSProvider{}
}
