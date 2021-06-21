package utils

import (
	v1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func InitJobRuntimeObjectByKind(kind string) runtime.Object {
	var (
		job runtime.Object
	)

	switch kind {
	case v1.TFJobKind:
		job = &v1.TFJob{}
	case v1.PyTorchJobKind:
		job = &v1.PyTorchJob{}
	case v1.XDLJobKind:
		job = &v1.XDLJob{}
	case v1.XGBoostJobKind:
		job = &v1.XGBoostJob{}
	}

	return job
}

func InitJobMetaObjectByKind(kind string) metav1.Object {
	var (
		job metav1.Object
	)

	switch kind {
	case v1.TFJobKind:
		job = &v1.TFJob{}
	case v1.PyTorchJobKind:
		job = &v1.PyTorchJob{}
	case v1.XDLJobKind:
		job = &v1.XDLJob{}
	case v1.XGBoostJobKind:
		job = &v1.XGBoostJob{}
	}

	return job

}

func RuntimeObjToMetaObj(obj runtime.Object) (metaObj metav1.Object, ok bool) {
	meta, ok := obj.(metav1.Object)
	return meta, ok
}
