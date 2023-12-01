package pytorch

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/openkruise/kruise/apis/apps/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/alibaba/kubedl/apis"
	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/metrics"
)

func TestRefreshStalePod(t *testing.T) {
	testCases := []struct {
		name                  string
		pod                   *v1.Pod
		crr                   *v1alpha1.ContainerRecreateRequest
		expectedPod           *v1.Pod
		worldSize, generation int64
	}{
		{
			name:        "pod with expected generation",
			pod:         newPod("default", "pod-1", 1, 2),
			expectedPod: newPod("default", "pod-1", 1, 2),
			worldSize:   2,
			generation:  1,
		},
		{
			name:        "stale pod with staled world size annotation value",
			pod:         newPod("default", "pod-2", 1, 2),
			expectedPod: newPod("default", "pod-2", 1, 3),
			worldSize:   3,
			generation:  2,
		},
		{
			name:        "stale pod and container restart has not finished",
			pod:         newPod("default", "pod-3", 1, 2),
			expectedPod: newPod("default", "pod-3", 1, 3),
			crr:         newContainerRecreateRequest("default", "pod-3", "1", v1alpha1.ContainerRecreateRequestPending),
			worldSize:   3,
			generation:  2,
		},
		{
			name:        "stale pod and container restart succeeded",
			pod:         newPod("default", "pod-4", 1, 3),
			expectedPod: newPod("default", "pod-4", 2, 3),
			crr:         newContainerRecreateRequest("default", "pod-4", "2", v1alpha1.ContainerRecreateRequestSucceeded),
			worldSize:   3,
			generation:  2,
		},
	}

	fakejob := &trainingv1alpha1.PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       trainingv1alpha1.PyTorchJobKind,
			APIVersion: trainingv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job",
			Namespace: "default",
			UID:       "123456",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_ = v1alpha1.AddToScheme(scheme.Scheme)
			fake := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			fakeRec := record.NewFakeRecorder(10)
			fakeR := PytorchJobReconciler{Client: fake, recorder: fakeRec,
				ctrl: job_controller.JobController{Client: fake, Recorder: fakeRec}}

			err := fake.Create(context.Background(), testCase.pod)
			if err != nil {
				t.Error(err)
			}
			if testCase.crr != nil {
				if err = fake.Create(context.Background(), testCase.crr.DeepCopy()); err != nil {
					t.Error(err)
				}
				_ = fake.Update(context.Background(), testCase.crr)
			}

			fakejob.SetGeneration(testCase.generation)

			_, err = fakeR.restartStalePod(fakejob, testCase.pod, testCase.worldSize, testCase.generation)
			if err != nil {
				t.Errorf("failed to refresh stale pod, err: %v", err)
			}

			p := v1.Pod{}
			err = fake.Get(context.Background(), types.NamespacedName{Namespace: testCase.pod.Namespace, Name: testCase.pod.Name}, &p)
			if err != nil {
				t.Error(err)
			}
			if p.Labels[apiv1.LabelGeneration] != testCase.expectedPod.Labels[apiv1.LabelGeneration] {
				t.Errorf("unexpected generation, expected: %v, got: %v", testCase.expectedPod.Labels[apiv1.LabelGeneration], p.Labels[apiv1.LabelGeneration])
			}
			if p.Annotations[AnnotationWorldSize] != testCase.expectedPod.Annotations[AnnotationWorldSize] {
				t.Errorf("unexpected worldsize, expected: %v, got: %v", testCase.expectedPod.Annotations[AnnotationWorldSize], p.Annotations[AnnotationWorldSize])
			}
		})
	}
}

func TestRefreshStaleService(t *testing.T) {
	testCases := []struct {
		generation  int64
		svc         v1.Service
		expectedSvc v1.Service
	}{
		{
			generation: 1,
			svc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type": "worker",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type": "worker",
					},
				},
			},
			expectedSvc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "1",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "1",
					},
				},
			},
		},
		{
			generation: 2,
			svc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "1",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "1",
					},
				},
			},
			expectedSvc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "2",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "2",
					},
				},
			},
		},
		{
			generation: 3,
			svc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "3",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "3",
					},
				},
			},
			expectedSvc: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-1",
					Namespace: "default",
					Labels: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "3",
					},
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"replica-type":        "worker",
						apiv1.LabelGeneration: "3",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		fake := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		fakeR := PytorchJobReconciler{Client: fake, recorder: record.NewFakeRecorder(10)}

		err := fake.Create(context.Background(), &testCase.svc)
		if err != nil {
			t.Error(err)
		}
		err = fakeR.refreshStaleService(&testCase.svc, testCase.generation)
		if err != nil {
			t.Error(err)
		}
		got := v1.Service{}
		err = fake.Get(context.Background(), types.NamespacedName{Namespace: testCase.svc.Namespace, Name: testCase.svc.Name}, &got)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(got.Labels, testCase.expectedSvc.Labels) {
			t.Errorf("unexpected service labels, expected: %+v, got: %+v", testCase.expectedSvc.Labels, got.Labels)
		}
		if !reflect.DeepEqual(got.Spec.Selector, testCase.expectedSvc.Spec.Selector) {
			t.Errorf("unexpected service selector, expected: %+v, got: %+v", testCase.expectedSvc.Spec.Selector, got.Spec.Selector)
		}
	}
}

func TestAddMasterWaiterForWorker(t *testing.T) {
	testCases := []struct {
		masterAddr      string
		initImage       string
		podSpec         v1.PodTemplateSpec
		expectedPodSpec v1.PodTemplateSpec
	}{
		{
			masterAddr: "addr-1",
			initImage:  "busybox",
			podSpec: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "main-image",
						},
					}},
			},
			expectedPodSpec: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Name:            "master-waiter",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Env: []v1.EnvVar{
								{
									Name:  "MASTER_ADDR",
									Value: "addr-1",
								},
							},
							Command: []string{"sh", "-c", "until ping -c1 $MASTER_ADDR >/dev/null 2>&1; do :; sleep 0.1; done;"},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "main-image",
						},
					}},
			},
		},
		{
			masterAddr: "addr-2",
			initImage:  "alpine",
			podSpec: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "main-image",
						},
					}},
			},
			expectedPodSpec: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Name:            "master-waiter",
							Image:           "alpine",
							ImagePullPolicy: v1.PullIfNotPresent,
							Env: []v1.EnvVar{
								{
									Name:  "MASTER_ADDR",
									Value: "addr-2",
								},
							},
							Command: []string{"sh", "-c", "until ping -c1 $MASTER_ADDR >/dev/null 2>&1; do :; sleep 0.1; done;"},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "main-image",
						},
					}},
			},
		},
	}

	for _, testCase := range testCases {
		err := AddMasterWaiterForWorker(&testCase.podSpec, InitContainerParam{
			MasterAddr:         testCase.masterAddr,
			InitContainerImage: testCase.initImage,
		})
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(testCase.podSpec, testCase.expectedPodSpec) {
			t.Errorf("unexpected pod template, expected: %+v, got: %+v", testCase.expectedPodSpec, testCase.podSpec)
		}
	}
}

func TestCheckpointIfNecessary(t *testing.T) {
	originNowFunc := nowFunc
	nowFunc = func() metav1.Time {
		fakeT, _ := time.Parse("2006-01-02 15:04:05", "2021-01-01 11:11:11")
		return metav1.NewTime(fakeT)
	}
	defer func() {
		nowFunc = originNowFunc
	}()

	testCases := []struct {
		name              string
		pytorchJob        *trainingv1alpha1.PyTorchJob
		expectedJob       *trainingv1alpha1.PyTorchJob
		pods              []*v1.Pod
		expectedPodsCount int
		expectedCompleted bool
	}{
		{
			name:        "no victim pods and just return",
			pytorchJob:  newPytorchJob("job-1", "default", nil, nil, 1),
			expectedJob: newPytorchJob("job-1", "default", nil, nil, 1),
			pods: []*v1.Pod{
				newPod("default", "pod-1", 1, 1),
				newPod("default", "pod-2", 1, 1),
			},
			expectedPodsCount: 2,
			expectedCompleted: true,
		},
		{
			name:       "first observe victim pod and trigger checkpoint",
			pytorchJob: newPytorchJob("job-2", "default", nil, nil, 1),
			expectedJob: newPytorchJob("job-2", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"status":"InProgress","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 1),
			pods: []*v1.Pod{
				newPod("default", "pod-3", 1, 1),
				newVictimPod("default", "pod-4", 1, 1),
			},
			expectedPodsCount: 2,
			expectedCompleted: false,
		},
		{
			name: "trigger first checkpoint and wait for first completion response",
			pytorchJob: newPytorchJob("job-3", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 1),
			expectedJob: newPytorchJob("job-3", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 1),
			pods: []*v1.Pod{
				newPod("default", "pod-5", 1, 1),
				newVictimPod("default", "pod-6", 1, 1),
			},
			expectedPodsCount: 2,
			expectedCompleted: false,
		},
		{
			name: "trigger first checkpoint and aimaster ack with completion response",
			pytorchJob: newPytorchJob("job-4", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"status":"InProgress","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":1,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 1),
			expectedJob: newPytorchJob("job-4", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"status":"Succeeded","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":1,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationReadyToStartWorker:         "true",
			}, 2),
			pods: []*v1.Pod{
				newPod("default", "pod-7", 1, 1),
				newVictimPod("default", "pod-8", 1, 1),
			},
			expectedPodsCount: 1,
			expectedCompleted: true,
		},
		{
			name: "trigger new checkpoint when observe new victim pods",
			pytorchJob: newPytorchJob("job-5", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":1,"status":"Succeeded","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":1,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 2),
			expectedJob: newPytorchJob("job-5", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":2,"status":"InProgress","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":1,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 2),
			pods: []*v1.Pod{
				newPod("default", "pod-9", 2, 1),
				newVictimPod("default", "pod-10", 2, 1),
			},
			expectedPodsCount: 2,
			expectedCompleted: false,
		},
		{
			name: "new checkpoint completed successfully and evict victims",
			pytorchJob: newPytorchJob("job-6", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":2,"status":"InProgress","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":2,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
			}, 2),
			expectedJob: newPytorchJob("job-6", "default", nil, map[string]string{
				AnnotationCheckpointRequestedVersion: `{"version":2,"status":"Succeeded","context":"pytorch job starts to request for checkpoint","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationCheckpointCompletedVersion: `{"version":2,"context":"","timestamp":"2021-01-01T11:11:11Z"}`,
				AnnotationReadyToStartWorker:         "true",
			}, 3),
			pods: []*v1.Pod{
				newPod("default", "pod-9", 2, 1),
				newVictimPod("default", "pod-10", 2, 1),
			},
			expectedPodsCount: 1,
			expectedCompleted: true,
		},
	}

	_ = apis.AddToScheme(scheme.Scheme)
	m := metrics.NewJobMetrics(trainingv1alpha1.PyTorchJobKind, fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build())
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			fakeRecorder := record.NewFakeRecorder(10)

			err := fakeClient.Create(context.Background(), testCase.pytorchJob)
			if err != nil {
				t.Errorf("client Create PyTrochJob, err: %v", err)
			}
			for _, p := range testCase.pods {
				if err = fakeClient.Create(context.Background(), p); err != nil {
					if err != nil {
						t.Errorf("client Create Pod, err: %v", err)
					}
				}
			}
			r := PytorchJobReconciler{
				scheme:   scheme.Scheme,
				Client:   fakeClient,
				recorder: fakeRecorder,
				ctrl:     job_controller.JobController{Metrics: m, PodControl: job_controller.NewPodControl(fakeClient, fakeRecorder)},
			}

			completed, err := r.CheckpointIfNecessary(testCase.pytorchJob, testCase.pods)
			if err != nil {
				t.Errorf("error when CheckpointIfNecessary, err: %v", err)
			}
			if completed != testCase.expectedCompleted {
				t.Errorf("unexpected completed result, expected: %v, actual: %v", testCase.expectedCompleted, completed)
			}
			pods := v1.PodList{}
			if err = fakeClient.List(context.Background(), &pods); err != nil {
				t.Errorf("failed to list pods, err: %v", err)
			}
			if len(pods.Items) != testCase.expectedPodsCount {
				t.Errorf("unexpected pod size, expected: %v, actual: %v", testCase.expectedPodsCount, len(pods.Items))
			}
			pj := trainingv1alpha1.PyTorchJob{}
			err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: testCase.pytorchJob.Namespace, Name: testCase.pytorchJob.Name}, &pj)
			if err != nil {
				t.Errorf("failed to get pytorch job, err: %v", err)
			}

			om1 := metav1.ObjectMeta{
				Labels:      pj.Labels,
				Annotations: pj.Annotations,
			}
			om2 := metav1.ObjectMeta{
				Labels:      testCase.expectedJob.Labels,
				Annotations: testCase.expectedJob.Annotations,
			}
			if !reflect.DeepEqual(om1, om2) {
				t.Errorf("unexpected pytorch job, expected: %v, actual: %v", om2, om1)
			}
		})
	}
}

func newPod(ns, name string, generation, worldSize int64) *v1.Pod {
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				apiv1.LabelGeneration: strconv.Itoa(int(generation)),
			},
			Annotations: map[string]string{
				AnnotationWorldSize: strconv.Itoa(int(worldSize)),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{},
			},
		},
	}

	return p
}

func newVictimPod(ns, name string, generation, worldSize int64) *v1.Pod {
	p := newPod(ns, name, generation, worldSize)
	p.Finalizers = append(p.Finalizers, apiv1.FinalizerPreemptProtector)
	now := metav1.Now()
	p.DeletionTimestamp = &now
	return p
}

func newContainerRecreateRequest(ns, name, generation string, phase v1alpha1.ContainerRecreateRequestPhase) *v1alpha1.ContainerRecreateRequest {
	crr := v1alpha1.ContainerRecreateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				apiv1.LabelGeneration: generation,
			},
		},
		Spec: v1alpha1.ContainerRecreateRequestSpec{
			PodName: name,
		},
		Status: v1alpha1.ContainerRecreateRequestStatus{
			Phase: phase,
		},
	}
	return &crr
}

func newPytorchJob(name, ns string, labels, annotations map[string]string, generation int64) *trainingv1alpha1.PyTorchJob {
	py := &trainingv1alpha1.PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PyTorchJob",
			APIVersion: "kubeflow.org/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
			Labels:      labels,
			Annotations: annotations,
			Generation:  generation,
		},
		Spec: trainingv1alpha1.PyTorchJobSpec{
			RunPolicy: apiv1.RunPolicy{},
			PyTorchReplicaSpecs: map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
				trainingv1alpha1.PyTorchReplicaTypeMaster: {
					Replicas: pointer.Int32Ptr(1),
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "pytorch",
								},
							},
						},
					},
				},
			},
		},
	}
	return py
}
