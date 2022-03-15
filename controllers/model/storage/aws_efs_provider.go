package storage

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
)

type AWSEfsProvider struct {
}

func (a *AWSEfsProvider) GetModelMountPath(mv *modelv1alpha1.Storage) string {
	panic("implement me")
}

func (a *AWSEfsProvider) AddModelVolumeToPodSpec(mv *modelv1alpha1.Storage, pod *corev1.PodTemplateSpec) {
	panic("implement me")
}

// The volumeHandle is of the form "[FileSystemId]:[Subpath]:[AccessPointId]"
// e.g. FilesystemId with subpath and access point Id:  fs-e8a95a42:/my/subpath:fsap-19f752f0068c22464.
// FilesystemId with access point Id:   				fs-e8a95a42::fsap-068c22f0246419f75
// FileSystemId with subpath: 	 						fs-e8a95a42:/dir1
//func (a *AWSEfsProvider) GetModelPath(model *modelv1alpha1.Storage) string {
//	// if no subpath
//	if !strings.Contains(model.AWSEfs.VolumeHandle, ":") {
//		return "/"
//	}
//	parts := strings.Split(model.AWSEfs.VolumeHandle, ":")
//	// only FilesystemId
//	if len(parts) == 1 {
//		return "/"
//	}
//	//
//	if parts[1] == "" {
//		return "/"
//	}
//	// the subpath
//	return parts[1]
//}

func (a *AWSEfsProvider) CreatePersistentVolume(storage *modelv1alpha1.Storage, pvName string) *corev1.PersistentVolume {

	// Get the volume attributes
	attributes := make(map[string]string, len(storage.AWSEfs.Attributes))
	for key, val := range storage.AWSEfs.Attributes {
		attributes[key] = val
	}

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
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:           "efs.csi.aws.com",
					VolumeHandle:     storage.AWSEfs.VolumeHandle,
					VolumeAttributes: attributes,
				},
			},
			StorageClassName: "",
		},
	}
	return pv
}

func NewAWSEfsProvider() StorageProvider {
	return &AWSEfsProvider{}
}
