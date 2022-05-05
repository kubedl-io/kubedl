package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	trainingv1alpha1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller/api/v1"
)

func init() {
	flag.StringVar(&name, "name", "pytorch-dist-sendrecv-example", "name of pytorchjob example")
	flag.StringVar(&namespace, "ns", "default", "namespace of pytorchjob example")
}

var (
	name, namespace string
)

func main() {
	flag.Parse()
	config := controllerruntime.GetConfigOrDie()
	trainingv1alpha1.AddToScheme(scheme.Scheme)

	cli, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		fmt.Println("new cli err:", err)
		return
	}

	pytorchJob := createNewPyTorchJobExample(namespace, name)
	if err = cli.Create(context.Background(), pytorchJob); err != nil {
		klog.Errorf("failed to create pytorch example, err: %v", err)
		os.Exit(1)
	}
	fmt.Printf("succeed to create pytorch job example [%+v]\n", *pytorchJob)
}

func createNewPyTorchJobExample(ns, name string) *trainingv1alpha1.PyTorchJob {
	return &trainingv1alpha1.PyTorchJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: trainingv1alpha1.PyTorchJobSpec{
			PyTorchReplicaSpecs: map[v1.ReplicaType]*v1.ReplicaSpec{
				trainingv1alpha1.PyTorchReplicaTypeMaster: {
					Replicas:      pointer.Int32Ptr(1),
					RestartPolicy: v1.RestartPolicyExitCode,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            trainingv1alpha1.PyTorchJobDefaultContainerName,
									Image:           "kubedl/pytorch-dist-example",
									ImagePullPolicy: corev1.PullAlways,
								},
							},
						},
					},
				},
				trainingv1alpha1.PyTorchReplicaTypeWorker: {
					Replicas:      pointer.Int32Ptr(2),
					RestartPolicy: v1.RestartPolicyExitCode,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            trainingv1alpha1.PyTorchJobDefaultContainerName,
									Image:           "kubedl/pytorch-dist-example",
									ImagePullPolicy: corev1.PullAlways,
								},
							},
						},
					},
				},
			},
		},
	}
}
