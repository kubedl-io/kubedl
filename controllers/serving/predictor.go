/*
Copyright 2021 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serving

import (
	"context"
	"fmt"

	modelv1alpha1 "github.com/alibaba/kubedl/apis/model/v1alpha1"
	"github.com/alibaba/kubedl/apis/serving/v1alpha1"
	"github.com/alibaba/kubedl/controllers/serving/framework"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// createNewPredictorDeploy creates a predictor deployment with containers to serve models.
func (ir *InferenceReconciler) createNewPredictorDeploy(inf *v1alpha1.Inference, modelVersion *modelv1alpha1.ModelVersion, predictor *v1alpha1.PredictorSpec) (*v1.Deployment, error) {
	predictorLabels := insertLabelsForPredictor(nil, inf, predictor)

	deploy := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      genPredictorName(inf, predictor),
			Namespace: inf.Namespace,
			Labels:    predictorLabels,
		},
	}

	// Set owner-reference to inference object.
	err := controllerutil.SetControllerReference(inf, &deploy, ir.scheme)
	if err != nil {
		return nil, err
	}

	modelMntPath := ""
	if modelVersion != nil && modelVersion.Name != "" {
		// 1. Init Container (that contains the model under /kubedl-model/) will be downloaded.
		// 2. A shared EmptyDir volume is mounted in InitContainer at "/mnt/kubedl-model"
		// 3. Move the models into shared volume: mv /kubedl-model/*  /mnt/kubedl-model
		// 4. The main serving container mounts the shared volume at "predictor.ModelPath" if specified or "/kubedl-model/<modelVersion-name>" if not.
		// 5. When serving, it loads the model from "predictor.ModelPath" if specified or "/kubedl-model/<modelVersion-name>" if not.
		sharedVolume := corev1.Volume{
			Name:         "kubedl-model-loader",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		}
		loader, err := ir.buildModelLoaderInitContainer(modelVersion, sharedVolume.Name, "/mnt/kubedl-model/")
		if err != nil {
			return nil, err
		}
		predictor.Template.Spec.Volumes = append(predictor.Template.Spec.Volumes, sharedVolume)
		predictor.Template.Spec.InitContainers = append(predictor.Template.Spec.InitContainers, *loader)

		modelMntPath = fmt.Sprintf("%s/%s", modelv1alpha1.DefaultModelPathInImage, modelVersion.Name)
		if predictor.ModelPath != nil && len(*predictor.ModelPath) > 0 {
			modelMntPath = *predictor.ModelPath
		}
		for ci := range predictor.Template.Spec.Containers {
			c := &predictor.Template.Spec.Containers[ci]
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				Name:      sharedVolume.Name,
				MountPath: modelMntPath,
			})
			c.Env = append(c.Env, corev1.EnvVar{Name: modelv1alpha1.KubeDLModelPath, Value: modelMntPath})
		}
	}

	// Setup predictor template with specific framework requirements.
	setter := framework.Setters[inf.Spec.Framework]
	if setter != nil {
		// Setup predictor template by its specific framework type.
		setter.SetSpec(&predictor.Template, modelVersion, modelMntPath)
	}

	selector := &metav1.LabelSelector{MatchLabels: predictorLabels}

	deploy.Spec = v1.DeploymentSpec{
		Replicas: predictor.Replicas,
		Selector: selector,
		Template: *predictor.Template.DeepCopy(),
		Strategy: v1.DeploymentStrategy{Type: v1.RollingUpdateDeploymentStrategyType},
	}
	deploy.Spec.Template.Labels = insertLabelsForPredictor(deploy.Spec.Template.Labels, inf, predictor)

	// Create service to route traffic forward to predictors with specific model version and
	// provide a load-balance frontend.
	if err = ir.createServiceForPredictor(inf, predictor); err != nil {
		return nil, err
	}

	if err = ir.client.Create(context.Background(), &deploy); err != nil {
		return nil, err
	}
	ir.recorder.Eventf(inf, corev1.EventTypeNormal, "PredictorDeploymentCreated",
		"Deployment %s for predictor successfully created, replicas: %d", deploy.Name, *deploy.Spec.Replicas)
	return &deploy, nil
}

func (ir *InferenceReconciler) createServiceForPredictor(inf *v1alpha1.Inference, predictor *v1alpha1.PredictorSpec) error {
	name := genPredictorName(inf, predictor)
	predictorLabels := insertLabelsForPredictor(nil, inf, predictor)
	svc := corev1.Service{}
	if err := ir.client.Get(context.Background(), types.NamespacedName{Namespace: inf.Namespace, Name: name}, &svc); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	// At this point, service not found and create a new one.

	httpPort := getContainerPort(predictor, v1alpha1.DefaultPredictorContainerHttpPortName, v1alpha1.DefaultPredictorContainerHttpPort)
	grpcPort := getContainerPort(predictor, v1alpha1.DefaultPredictorContainerGrpcPortName, v1alpha1.DefaultPredictorContainerGrpcPort)

	svc = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: inf.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: predictorLabels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       v1alpha1.DefaultPredictorContainerHttpPort,
					TargetPort: intstr.IntOrString{IntVal: httpPort},
				},
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       v1alpha1.DefaultPredictorContainerGrpcPort,
					TargetPort: intstr.IntOrString{IntVal: grpcPort},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(inf, &svc, ir.scheme); err != nil {
		return err
	}

	return ir.client.Create(context.Background(), &svc)
}

func getContainerPort(predictor *v1alpha1.PredictorSpec, portName string, defaultVal int32) int32 {
	for ci := range predictor.Template.Spec.Containers {
		container := &predictor.Template.Spec.Containers[ci]
		for pi := range container.Ports {
			if container.Ports[pi].Name == portName {
				return container.Ports[pi].ContainerPort
			}
		}
	}
	return defaultVal
}
