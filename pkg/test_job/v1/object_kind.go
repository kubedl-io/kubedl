package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testObjectKind struct{}

func (t *testObjectKind) SetGroupVersionKind(kind schema.GroupVersionKind) {}

func (t *testObjectKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: GroupVersion}
}

func (in *TestJob) GetObjectKind() schema.ObjectKind {
	var _ schema.ObjectKind = &testObjectKind{}

	TestObjectKind := &testObjectKind{}
	return TestObjectKind
}
