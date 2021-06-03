package coscheduler

import (
	"encoding/json"
	tfv1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/features"
	"github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"

	"github.com/alibaba/kubedl/apis"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCoscheduler_CreateGang(t *testing.T) {
	tfGVK := schema.GroupVersionKind{
		Group:   "kubeflow.org",
		Version: "v1",
		Kind:    "TFJob",
	}
	testCases := []struct {
		name     string
		job      metav1.Object
		podGroup v1alpha1.PodGroup
	}{
		{
			name: "input tf job with 1 ps and 1 worker and no run policy",
			job:  createJob("job-1", 1, nil),
			podGroup: createGang("job-1", "", "job-1", "", map[string]int{
				"job-1-ps": 1, "job-1-worker-0": 1,
			}, metav1.NewControllerRef(createJob("job-1", 1, nil), tfGVK)),
		},
		{
			name: "input tf job with 1 ps and 3 worker and no run policy",
			job:  createJob("job-2", 3, nil),
			podGroup: createGang("job-1", "", "job-1", "", map[string]int{
				"job-2-ps": 1, "job-2-worker": 2, "job-2-worker-0": 1,
			}, metav1.NewControllerRef(createJob("job-2", 3, nil), tfGVK)),
		},
	}

	dagEnabled := features.KubeDLFeatureGates.Enabled(features.DAGScheduling)
	defer features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): dagEnabled})

	for _, testCase := range testCases {
		scheme := runtime.NewScheme()
		_ = apis.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)
		us := kubeCoscheduler{client: fake.NewFakeClientWithScheme(scheme)}

		var (
			specs map[v1.ReplicaType]*v1.ReplicaSpec
		)

		switch testCase.job.(type) {
		case *tfv1.TFJob:
			specs = testCase.job.(*tfv1.TFJob).Spec.TFReplicaSpecs
		default:
			t.Errorf("unknown test job type: %v", testCase.job)
			return
		}

		podGroup, err := us.CreateGang(testCase.job, specs)
		if err != nil {
			t.Errorf("failed to create podGroup, err: %v", err)
		}

		podGroup, err = us.GetGang(types.NamespacedName{Namespace: testCase.job.GetNamespace(), Name: testCase.job.GetName()})
		if err != nil {
			t.Errorf("failed to get latest podGroup, err: %v", err)
		}
		testCase.podGroup.TypeMeta = metav1.TypeMeta{Kind: "GangList", APIVersion: "scheduling.alibabacloud.com/v1alpha1"}

		if !deepEqual(podGroup, &testCase.podGroup) {
			t.Errorf("unexpected podGroup result, expected: %+v, got: %+v", &testCase.podGroup, podGroup)
		}
	}
}

func TestUniScheduler_BindPodToGang(t *testing.T) {
	testCases := []struct {
		name         string
		job          metav1.Object
		podGroup     v1alpha1.PodGroup
		podSpec      *corev1.PodTemplateSpec
		rtype, index string
		expectedSpec *corev1.PodTemplateSpec
		dag          bool
	}{
		{
			name:     "bind pod for tf job ps",
			job:      createJob("job-1", 1, nil),
			podGroup: createGang("job-1", "12345", "job-1", "", nil, nil),
			podSpec:  &corev1.PodTemplateSpec{},
			rtype:    "ps",
			index:    "0",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
					{
						//APIVersion:         "scheduling.alibabacloud.com/v1alpha1",
						//Kind:               "podGroup",
						//Name:               "job-1",
						//UID:                "12345",
						//Controller:         pointer.BoolPtr(false),
						//BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
					Labels: map[string]string{
						//v1alpha1.LabelGangName:   "job-1",
						//v1alpha1.LabelBundleName: "job-1-ps",
					},
				},
			},
		},
		{
			name:     "bind pod for tf job worker(total 1, index 0)",
			job:      createJob("job-2", 1, nil),
			podGroup: createGang("job-2", "12345", "job-2", "", nil, nil),
			podSpec:  &corev1.PodTemplateSpec{},
			rtype:    "worker",
			index:    "0",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
					{
						//APIVersion:         "scheduling.alibabacloud.com/v1alpha1",
						//Kind:               "podGroup",
						//Name:               "job-2",
						//UID:                "12345",
						//Controller:         pointer.BoolPtr(false),
						//BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
					Labels: map[string]string{
						//v1alpha1.LabelGangName:   "job-2",
						//v1alpha1.LabelBundleName: "job-2-worker-0",
					},
				},
			},
		},
		{
			name:     "bind pod for tf job worker(total 3, index 1)",
			job:      createJob("job-3", 3, nil),
			podGroup: createGang("job-3", "12345", "job-3", "", nil, nil),
			podSpec:  &corev1.PodTemplateSpec{},
			rtype:    "worker",
			index:    "1",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
					{
						//APIVersion:         "scheduling.alibabacloud.com/v1alpha1",
						//Kind:               "podGroup",
						//Name:               "job-3",
						//UID:                "12345",
						//Controller:         pointer.BoolPtr(false),
						//BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
					Labels: map[string]string{
						//v1alpha1.LabelGangName:   "job-3",
						//v1alpha1.LabelBundleName: "job-3-worker",
					},
				},
			},
		},
		{
			name:     "bind pod for pytorch job worker(total 1, index 0)",
			job:      createJob("job-4", 1, nil),
			podGroup: createGang("job-4", "12345", "job-4", "", nil, nil),
			podSpec:  &corev1.PodTemplateSpec{},
			rtype:    "worker",
			index:    "0",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
					{
						//APIVersion:         "scheduling.alibabacloud.com/v1alpha1",
						//Kind:               "podGroup",
						//Name:               "job-4",
						//UID:                "12345",
						//Controller:         pointer.BoolPtr(false),
						//BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
					Labels: map[string]string{
						//v1alpha1.LabelGangName:   "job-4",
						//v1alpha1.LabelBundleName: "job-4-worker",
					},
				},
			},
		},
		{
			name:     "bind pod for pytorch job worker(total 3, index 1)",
			job:      createJob("job-5", 3, nil),
			podGroup: createGang("job-5", "12345", "job-5", "", nil, nil),
			podSpec:  &corev1.PodTemplateSpec{},
			rtype:    "worker",
			index:    "1",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
					{
						//APIVersion:         "scheduling.alibabacloud.com/v1alpha1",
						//Kind:               "podGroup",
						//Name:               "job-5",
						//UID:                "12345",
						//Controller:         pointer.BoolPtr(false),
						//BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
					Labels: map[string]string{
						//LabelGangName:   "job-5",
						//v1.LabelBundleName: "job-5-worker",
					},
				},
			},
		},
	}

	dagEnabled := features.KubeDLFeatureGates.Enabled(features.DAGScheduling)
	defer features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): dagEnabled})

	for _, testCase := range testCases {
		scheme := runtime.NewScheme()
		_ = apis.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)
		us := kubeCoscheduler{client: fake.NewFakeClientWithScheme(scheme)}
		// only for test
		features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): testCase.dag})
		//for i := range testCase.podGroup.Items {
		//	testCase.podGroup.Items[i].TypeMeta = metav1.TypeMeta{APIVersion: "scheduling.alibabacloud.com/v1alpha1", Kind: "podGroup"}
		//}
		// err := us.BindPodToGang(testCase.job, testCase.podSpec, &testCase.podGroup, testCase.rtype, testCase.index)
		err := us.BindPodToGang(testCase.podSpec, &testCase.podGroup)
		if err != nil {
			t.Errorf("failed to bind pod to podGroup, err: %v", err)
		}
		if !deepEqual(testCase.expectedSpec, testCase.podSpec) {
			t.Errorf("unexpected bind result, expected: %+v, got: %+v", testCase.expectedSpec, testCase.podSpec)
		}
	}
}

func deepEqual(obj1, obj2 interface{}) bool {
	if reflect.DeepEqual(obj1, obj2) {
		return true
	}
	dump1, err := json.Marshal(obj1)
	if err != nil {
		return false
	}
	dump2, err := json.Marshal(obj2)
	if err != nil {
		return false
	}
	return string(dump1) == string(dump2)
}

func createJob(jobName string, workerReplicas int32, runPolicy *v1.RunPolicy) *tfv1.TFJob {
	job := &tfv1.TFJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeflow.org/v1",
			Kind:       "TFJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},

		Spec: tfv1.TFJobSpec{
			TFReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				"PS": {
					Replicas:      pointer.Int32Ptr(1),
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "tensorflow",
									Image: "kubedl/tf-mnist-with-summaries:1.0",
								},
							},
						},
					},
				},
				"Worker": {
					Replicas:      &workerReplicas,
					RestartPolicy: "Never",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "tensorflow",
									Image: "kubedl/tf-mnist-with-summaries:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	if runPolicy != nil {
		job.Spec.RunPolicy = *runPolicy
	}
	return job
}

func createGang(name, uid, jobName, rtype string, bundles map[string]int, owner *metav1.OwnerReference) v1alpha1.PodGroup {
	podGroup := v1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			ResourceVersion: "1",
			Labels:          map[string]string{},
		},
	}
	if uid != "" {
		podGroup.UID = types.UID(uid)
	}
	if owner != nil {
		podGroup.OwnerReferences = append(podGroup.OwnerReferences, *owner)
	}
	//if jobName != "" {
	//	podGroup.Labels[v1.JobNameLabel] = jobName
	//}
	//if rtype != "" {
	//	podGroup.Labels[v1.ReplicaTypeLabel] = rtype
	//}

	//if bundles != nil {
	//	for name, min := range bundles {
	//		podGroup.Spec.Bundles = append(podGroup.Spec.Bundles, v1alpha1.BundleSpec{
	//			Name:              name,
	//			MinRequiredNumber: int32(min),
	//		})
	//	}
	//	sort.SliceStable(podGroup.Spec.Bundles, func(i, j int) bool {
	//		return podGroup.Spec.Bundles[i].Name < podGroup.Spec.Bundles[j].Name
	//	})
	//}
	return podGroup
}
