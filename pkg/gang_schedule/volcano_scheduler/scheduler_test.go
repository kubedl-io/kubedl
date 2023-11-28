package volcano_scheduler

import (
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/alibaba/kubedl/apis"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/features"
	v1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func TestVolcano_CreateGang(t *testing.T) {
	tfGVK := schema.GroupVersionKind{
		Group:   trainingv1alpha1.GroupVersion.Group,
		Version: trainingv1alpha1.GroupVersion.Version,
		Kind:    trainingv1alpha1.TFJobKind,
	}
	pytorchGVK := schema.GroupVersionKind{
		Group:   trainingv1alpha1.GroupVersion.Group,
		Version: trainingv1alpha1.GroupVersion.Version,
		Kind:    trainingv1alpha1.PyTorchJobKind,
	}
	testCases := []struct {
		name       string
		job        metav1.Object
		podgrpoups v1beta1.PodGroupList
		dag        bool
	}{
		{
			name:       "input tf job with 1 ps and 1 worker and no run policy",
			job:        createTFJob("job-1", 1, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-1", "", "job-1", "", 2, metav1.NewControllerRef(createTFJob("job-1", 1, nil), tfGVK))}},
		},
		{
			name:       "input tf job with 1 ps and 3 worker and no run policy",
			job:        createTFJob("job-2", 3, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-2", "", "job-2", "", 4, metav1.NewControllerRef(createTFJob("job-2", 3, nil), tfGVK))}},
		},
		{
			name: "input tf job with 1 ps and 3 worker and min-available(2) in run policy",
			job: createTFJob("job-3", 3, &v1.RunPolicy{SchedulingPolicy: &v1.SchedulingPolicy{
				MinAvailable: pointer.Int32Ptr(2)}}),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-3", "", "job-3", "", 2,
				metav1.NewControllerRef(createTFJob("job-3", 3, &v1.RunPolicy{SchedulingPolicy: &v1.SchedulingPolicy{
					MinAvailable: pointer.Int32Ptr(2)}}), tfGVK))}},
		},
		{
			name: "input pytorch job with 1 worker and no run policy",
			job:  createPytorchJob("job-4", 1, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-4", "", "job-4", "", 1, metav1.NewControllerRef(
				createTFJob("job-4", 1, nil), pytorchGVK))}},
		},
		{
			name: "input pytorch job with 3 worker and no run policy",
			job:  createPytorchJob("job-5", 3, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-5", "", "job-5", "", 3, metav1.NewControllerRef(
				createPytorchJob("job-5", 3, nil), pytorchGVK))}},
		},
		{
			name: "input pytorch job with 3 worker and min-available(2) in run policy",
			job: createPytorchJob("job-6", 3, &v1.RunPolicy{SchedulingPolicy: &v1.SchedulingPolicy{
				MinAvailable: pointer.Int32Ptr(2)}}),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-6", "", "job-6", "", 2, metav1.NewControllerRef(
				createTFJob("job-6", 3, &v1.RunPolicy{SchedulingPolicy: &v1.SchedulingPolicy{
					MinAvailable: pointer.Int32Ptr(2)}}), pytorchGVK))}},
		},
		{
			name: "input tf job with ps+worker and dag enabled.",
			job:  createTFJob("job-7", 1, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{
				createPodGroup("job-7-ps", "", "job-7", "ps", 1, metav1.NewControllerRef(createTFJob("job-7", 1, nil), tfGVK)),
				createPodGroup("job-7-worker", "", "job-7", "worker", 1, metav1.NewControllerRef(createTFJob("job-7", 1, nil), tfGVK)),
			}},
			dag: true,
		},
		{
			name: "input pytorch job with worker only and dag enabled.",
			job:  createPytorchJob("job-8", 3, nil),
			podgrpoups: v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-8-worker", "", "job-8", "worker", 3, metav1.NewControllerRef(
				createPytorchJob("job-8", 3, nil), pytorchGVK))}},
			dag: true,
		},
	}

	dagEnabled := features.KubeDLFeatureGates.Enabled(features.DAGScheduling)
	defer func() {
		_ = features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): dagEnabled})
	}()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = apis.AddToScheme(scheme)
			_ = v1beta1.AddToScheme(scheme)
			us := volcanoScheduler{client: fake.NewClientBuilder().WithScheme(scheme).Build()}

			var (
				specs     map[v1.ReplicaType]*v1.ReplicaSpec
				runPolicy *v1.RunPolicy
			)

			switch testCase.job.(type) {
			case *trainingv1alpha1.TFJob:
				specs = testCase.job.(*trainingv1alpha1.TFJob).Spec.TFReplicaSpecs
				runPolicy = &testCase.job.(*trainingv1alpha1.TFJob).Spec.RunPolicy
			case *trainingv1alpha1.PyTorchJob:
				specs = testCase.job.(*trainingv1alpha1.PyTorchJob).Spec.PyTorchReplicaSpecs
				runPolicy = &testCase.job.(*trainingv1alpha1.PyTorchJob).Spec.RunPolicy
			default:
				t.Errorf("unknown test job type: %v", testCase.job)
				return
			}

			// only for test
			_ = features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): testCase.dag})

			_, err := us.CreateGang(testCase.job, specs, runPolicy.SchedulingPolicy)
			if err != nil {
				t.Errorf("failed to create podgrpoups, err: %v", err)
			}

			gang, err := us.GetGang(types.NamespacedName{Namespace: testCase.job.GetNamespace(), Name: testCase.job.GetName()})
			if err != nil {
				t.Errorf("failed to get latest podgrpoups, err: %v", err)
			}
			testCase.podgrpoups.TypeMeta = metav1.TypeMeta{Kind: "PodGroupList", APIVersion: "scheduling.volcano.sh/v1beta1"}

			sort.SliceStable(gang.(*v1beta1.PodGroupList).Items, func(i, j int) bool {
				return gang.(*v1beta1.PodGroupList).Items[i].Name < gang.(*v1beta1.PodGroupList).Items[j].Name
			})
			sort.SliceStable(testCase.podgrpoups.Items, func(i, j int) bool {
				return testCase.podgrpoups.Items[i].Name < testCase.podgrpoups.Items[j].Name
			})

			if !equality.Semantic.DeepEqual(gang, &testCase.podgrpoups) {
				t.Errorf("unexpected podgrpoups result, expected: %+v, got: %+v", &testCase.podgrpoups, gang)
			}
		})
	}
}

func TestVolcano_BindPodToGang(t *testing.T) {
	testCases := []struct {
		name         string
		job          metav1.Object
		gang         v1beta1.PodGroupList
		podSpec      *corev1.PodTemplateSpec
		rtype        string
		expectedSpec *corev1.PodTemplateSpec
		dag          bool
	}{
		{
			name:    "bind pod for tf job ps",
			job:     createTFJob("job-1", 1, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-1", "12345", "job-1", "", 1, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "ps",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-1"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-1",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
		},
		{
			name:    "bind pod for tf job worker(total 1, index 0)",
			job:     createTFJob("job-2", 1, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-2", "12345", "job-2", "", 1, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "worker",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-2"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-2",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
		},
		{
			name:    "bind pod for tf job worker(total 3, index 1)",
			job:     createTFJob("job-3", 3, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-3", "12345", "job-3", "", 3, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "worker",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-3"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-3",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
		},
		{
			name:    "bind pod for pytorch job worker(total 1, index 0)",
			job:     createPytorchJob("job-4", 1, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-4", "12345", "job-4", "", 1, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "worker",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-4"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-4",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
		},
		{
			name:    "bind pod for pytorch job worker(total 3, index 1)",
			job:     createPytorchJob("job-5", 3, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-5", "12345", "job-5", "", 3, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "worker",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-5"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-5",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
		},
		{
			name:    "bind pod for tf job ps and dag enabled",
			job:     createTFJob("job-6", 1, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-6-ps", "12345", "job-6", "ps", 1, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "ps",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-6-ps"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-6-ps",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
			dag: true,
		},
		{
			name:    "bind pod for tf job worker and dag enabled",
			job:     createTFJob("job-7", 1, nil),
			gang:    v1beta1.PodGroupList{Items: []v1beta1.PodGroup{createPodGroup("job-7-worker", "12345", "job-7", "worker", 1, nil)}},
			podSpec: &corev1.PodTemplateSpec{},
			rtype:   "worker",
			expectedSpec: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{v1beta1.KubeGroupNameAnnotationKey: "job-7-worker"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "scheduling.volcano.sh/v1beta1",
							Kind:               "PodGroup",
							Name:               "job-7-worker",
							UID:                "12345",
							Controller:         pointer.BoolPtr(false),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					}},
			},
			dag: true,
		},
	}

	dagEnabled := features.KubeDLFeatureGates.Enabled(features.DAGScheduling)
	defer func() {
		_ = features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): dagEnabled})
	}()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = apis.AddToScheme(scheme)
			_ = v1beta1.AddToScheme(scheme)
			us := volcanoScheduler{client: fake.NewClientBuilder().WithScheme(scheme).Build()}
			// only for test
			_ = features.KubeDLFeatureGates.SetFromMap(map[string]bool{string(features.DAGScheduling): testCase.dag})
			for i := range testCase.gang.Items {
				testCase.gang.Items[i].TypeMeta = metav1.TypeMeta{APIVersion: "scheduling.volcano.sh/v1beta1", Kind: "PodGroup"}
			}
			err := us.BindPodToGang(testCase.job, testCase.podSpec, &testCase.gang, testCase.rtype)
			if err != nil {
				t.Errorf("failed to bind pod to podgrpoups, err: %v", err)
			}
			if !equality.Semantic.DeepEqual(testCase.expectedSpec, testCase.podSpec) {
				t.Errorf("unexpected bind result, expected: %+v, got: %+v", testCase.expectedSpec, testCase.podSpec)
			}
		})
	}
}

func createTFJob(jobName string, workerReplicas int32, runPolicy *v1.RunPolicy) *trainingv1alpha1.TFJob {
	job := &trainingv1alpha1.TFJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: trainingv1alpha1.GroupVersion.String(),
			Kind:       trainingv1alpha1.TFJobKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},

		Spec: trainingv1alpha1.TFJobSpec{
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

func createPytorchJob(jobName string, workerReplicas int32, runPolicy *v1.RunPolicy) *trainingv1alpha1.PyTorchJob {
	job := &trainingv1alpha1.PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: trainingv1alpha1.GroupVersion.String(),
			Kind:       trainingv1alpha1.PyTorchJobKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},

		Spec: trainingv1alpha1.PyTorchJobSpec{
			PyTorchReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
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

func createPodGroup(name, uid, jobName, rtype string, minMember int32, owner *metav1.OwnerReference) v1beta1.PodGroup {
	empty := make(corev1.ResourceList)
	pg := v1beta1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "default",
			ResourceVersion: "1",
			Labels:          map[string]string{},
		},
		Spec: v1beta1.PodGroupSpec{MinMember: minMember, MinResources: &empty},
	}
	if uid != "" {
		pg.UID = types.UID(uid)
	}
	if owner != nil {
		pg.OwnerReferences = append(pg.OwnerReferences, *owner)
	}
	if jobName != "" {
		pg.Labels[v1.LabelGangSchedulingJobName] = jobName
	}
	if rtype != "" {
		pg.Labels[v1.ReplicaTypeLabel] = rtype
	}
	return pg
}
