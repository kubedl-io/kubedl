package handlers

import (
	"encoding/json"
	"path"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/tensorboard"
	"github.com/alibaba/kubedl/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type preSubmitHook func(job runtime.Object)

func tfJobPreSubmitAutoConvertReplicas(job runtime.Object) {
	if job == nil {
		return
	}
	tfJob, ok := job.(*v1.TFJob)
	if !ok {
		return
	}

	totalReplicas := k8sutil.GetTotalReplicas(tfJob.Spec.TFReplicaSpecs)
	if tbSpec, ok := tfJob.Spec.TFReplicaSpecs[apiv1.ReplicaTypeTensorBoard]; ok {
		if tbSpec.Replicas != nil {
			totalReplicas -= *tbSpec.Replicas
		} else {
			totalReplicas -= 1
		}
	}
	_, workerExist := tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeWorker]
	_, chiefExist := tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeChief]
	if totalReplicas == 1 && workerExist && !chiefExist {
		workerSpec := tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeWorker]
		tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeChief] = workerSpec.DeepCopy()
		delete(tfJob.Spec.TFReplicaSpecs, v1.TFReplicaTypeWorker)
	}
}

func tfJobPreSubmitTensorBoardDefaults(job runtime.Object) {
	if job == nil {
		return
	}
	tfJob, ok := job.(*v1.TFJob)
	if !ok {
		return
	}
	tbConfig, ok := tfJob.Annotations[apiv1.AnnotationTensorBoardConfig]
	if !ok {
		return
	}
	tb := tensorboard.TensorBoard{}
	err := json.Unmarshal([]byte(tbConfig), &tb)
	if err != nil {
		return
	}
	if tb.Image == nil {
		imgCfg := NewKubeDLHandler().GetImageConfig()
		if len(imgCfg.TFCpuImages) == 0 {
			return
		}
		spec, ok := tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeMaster]
		if !ok {
			spec, ok = tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeChief]
			if !ok {
				spec = tfJob.Spec.TFReplicaSpecs[v1.TFReplicaTypeWorker]
			}
		}

		isTFV1 := func(image string) bool {
			version := strings.Split(image, ":")[1]
			return strings.HasPrefix(version, "1.")
		}

		image := getMainContainerImage(spec, v1.TFJobDefaultContainerName)
		isTFV1Img := isTFV1(image)
		tbImage := imgCfg.TFCpuImages[0]
		if len(image) > 0 {
			for _, kubedlImg := range imgCfg.TFCpuImages {
				if (isTFV1Img && isTFV1(kubedlImg)) || (!isTFV1Img && !isTFV1(kubedlImg)) {
					tbImage = kubedlImg
					break
				}
			}
		}
		tb.Image = pointer.StringPtr(tbImage)
	}
	presubmitTensorBoardDefaults(tfJob, tb)
}

func presubmitTensorBoardDefaults(metaObj metav1.Object, tb tensorboard.TensorBoard) {
	if tb.TTLSecondsAfterJobFinished == 0 {
		tb.TTLSecondsAfterJobFinished = 60 * 60
	}
	if tb.Ingress != nil && tb.Ingress.PathPrefix == nil {
		tb.Ingress.PathPrefix = pointer.StringPtr(path.Join("/", metaObj.GetNamespace(), metaObj.GetName()))
	}

	if tb.UpdateTimestamp.IsZero() {
		tb.UpdateTimestamp = metav1.Now()
	}
	configBytes, _ := json.Marshal(&tb)
	anno := metaObj.GetAnnotations()
	anno[apiv1.AnnotationTensorBoardConfig] = string(configBytes)
	metaObj.SetAnnotations(anno)
}

func pytorchJobPreSubmitAutoConvertReplicas(job runtime.Object) {
	if job == nil {
		return
	}
	pytorchJob, ok := job.(*v1.PyTorchJob)
	if !ok {
		return
	}

	var (
		workerExist, masterExist       bool
		workerReplicas, masterReplicas int32
	)

	_, workerExist = pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeWorker]
	_, masterExist = pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeMaster]

	if workerExist && pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeWorker].Replicas != nil {
		workerReplicas = *pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeWorker].Replicas
	}
	if masterExist && pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeMaster].Replicas != nil {
		masterReplicas = *pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeMaster].Replicas
	}

	if masterReplicas == 0 && workerReplicas > 0 {
		workerSpec := pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeWorker]
		pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeMaster] = workerSpec.DeepCopy()
		pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeMaster].Replicas = Int32(1)
		workerReplicas = workerReplicas - 1
	}

	if workerReplicas <= 0 {
		delete(pytorchJob.Spec.PyTorchReplicaSpecs, v1.PyTorchReplicaTypeWorker)
	} else if workerReplicas > 0 {
		pytorchJob.Spec.PyTorchReplicaSpecs[v1.PyTorchReplicaTypeWorker].Replicas = Int32(workerReplicas)
	}
}

func pytorchJobPreSubmitTensorBoardDefaults(job runtime.Object) {
	if job == nil {
		return
	}
	pytorchJob, ok := job.(*v1.PyTorchJob)
	if !ok {
		return
	}
	tbConfig, ok := pytorchJob.Annotations[apiv1.AnnotationTensorBoardConfig]
	if !ok {
		return
	}
	tb := tensorboard.TensorBoard{}
	err := json.Unmarshal([]byte(tbConfig), &tb)
	if err != nil {
		return
	}
	/*
		if tb.Image == nil {
			dlcCfg, err := NewDLCHandler().GetDLCConfig()
			if err != nil {
				return
			}
			if len(dlcCfg.TFCpuImages) == 0 {
				return
			}
			tbImage := dlcCfg.TFCpuImages[0]
			tb.Image = pointer.StringPtr(tbImage)
		}
	*/

	presubmitTensorBoardDefaults(pytorchJob, tb)
}

func getMainContainerImage(spec *apiv1.ReplicaSpec, main string) string {
	if spec == nil {
		return ""
	}
	for i := range spec.Template.Spec.Containers {
		if spec.Template.Spec.Containers[i].Name == main {
			return spec.Template.Spec.Containers[i].Image
		}
	}
	return ""
}

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}
