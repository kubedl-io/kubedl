package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	v1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
)

func InitJobRuntimeObjectByKind(kind string) client.Object {
	var (
		object client.Object
	)

	switch kind {
	case v1.TFJobKind:
		object = &v1.TFJob{}
	case v1.PyTorchJobKind:
		object = &v1.PyTorchJob{}
	case v1.XDLJobKind:
		object = &v1.XDLJob{}
	case v1.XGBoostJobKind:
		object = &v1.XGBoostJob{}
	case v1alpha1.NotebookKind:
		object = &v1alpha1.Notebook{}
	}

	return object
}

func RuntimeObjToMetaObj(obj runtime.Object) (metaObj metav1.Object, ok bool) {
	meta, ok := obj.(metav1.Object)
	return meta, ok
}
