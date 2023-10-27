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
	"reflect"

	beta1 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"

	"github.com/alibaba/kubedl/apis/model/v1alpha1"
	servingv1alpha1 "github.com/alibaba/kubedl/apis/serving/v1alpha1"
	"github.com/alibaba/kubedl/cmd/options"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	defaultIstioGatewayName = "kubedl-serving-gateway"
)

func NewInferenceReconciler(mgr ctrl.Manager, _ options.JobControllerConfiguration) *InferenceReconciler {
	r := &InferenceReconciler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("inference-controller"),
	}

	return r
}

// InferenceReconciler reconciles a Inference object
type InferenceReconciler struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func (ir *InferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("InferenceController", mgr, controller.Options{
		Reconciler:              ir,
		MaxConcurrentReconciles: options.CtrlConfig.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	// Watch inference owner resource.
	if err = c.Watch(&source.Kind{Type: &servingv1alpha1.Inference{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &v1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &servingv1alpha1.Inference{}}); err != nil {
		return err
	}
	return nil
}

// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.kubedl.io,resources=inferences,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.kubedl.io,resources=inferences/status,verbs=get;update;patch

func (ir *InferenceReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	inference := servingv1alpha1.Inference{}
	err := ir.client.Get(context.Background(), req.NamespacedName, &inference)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("inference instance %s not found, may have been deleted.", req.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	oldStatus := inference.Status.DeepCopy()

	klog.Infof("reconcile for inference instance: %s, status: %v", req.String(), oldStatus)

	// 1) Sync entrypoint service to access inference service for users.
	if err = ir.syncServiceForInference(&inference); err != nil {
		klog.Errorf("failed to sync service for inference, err: %v", err)
		return ctrl.Result{}, err
	}

	// 2) Sync each predictor to deploy containers mounted with specific model.
	for pi := range inference.Spec.Predictors {
		predictor := &inference.Spec.Predictors[pi]
		result, err := ir.syncPredictor(&inference, pi, predictor)
		if err != nil {
			klog.Errorf("failed to sync each predictor of inference[%s], err: %v", inference.Name, err)
			return result, err
		}
	}

	// 3) If inference serves multiple model version simultaneously and canary policy has been set,
	// serving traffic will be distributed with different ratio and routes to backend service.
	if len(inference.Spec.Predictors) > 1 {
		if err = ir.syncTrafficDistribution(&inference); err != nil {
			klog.Errorf("failed to sync traffic distribution of inference[%s], err: %v", inference.Name, err)
			return ctrl.Result{}, err
		}
	}

	// 4) Compare status before-and-after reconciling and update changes to cluster.
	if !reflect.DeepEqual(oldStatus, &inference.Status) {
		if err = ir.client.Status().Update(context.Background(), &inference); err != nil {
			if errors.IsConflict(err) {
				// retry later when update operation violates with etcd concurrency control.
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// syncPredictor syncs each predictor, deploy it with target modelversion and handles traffic distribution
// with canary policy.
// TODO: support autoscaling and batching for inference backends.
func (ir *InferenceReconciler) syncPredictor(inf *servingv1alpha1.Inference, index int, predictor *servingv1alpha1.PredictorSpec) (ctrl.Result, error) {
	klog.Infof("start to sync predictor-[%d], name: %s, model version: %s", index, predictor.Name, predictor.ModelVersion)

	var (
		modelVersion = v1alpha1.ModelVersion{}
		err          error
	)

	if predictor.ModelVersion != "" {
		err := ir.client.Get(context.Background(), types.NamespacedName{Namespace: inf.Namespace, Name: predictor.ModelVersion}, &modelVersion)
		if err != nil {
			return ctrl.Result{}, err
		}

		if modelVersion.Status.ImageBuildPhase != v1alpha1.ImageBuildSucceeded {
			klog.Infof("predictor model version has not been successfully built yet, current build phase %s", modelVersion.Status.ImageBuildPhase)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	deploy := &v1.Deployment{}
	err = ir.client.Get(context.Background(), types.NamespacedName{Namespace: inf.Namespace, Name: genPredictorName(inf, predictor)}, deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			deploy, err = ir.createNewPredictorDeploy(inf, &modelVersion, predictor)
			if err != nil {
				klog.Errorf("failed to create new predictor deployment, err: %v", err)
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	endpoint := svcHostForPredictor(inf, predictor)
	if psLen := len(inf.Status.PredictorStatuses); psLen == 0 || psLen < index {
		inf.Status.PredictorStatuses = append(inf.Status.PredictorStatuses, servingv1alpha1.PredictorStatus{
			Name:          predictor.Name,
			Replicas:      deploy.Status.Replicas,
			ReadyReplicas: deploy.Status.ReadyReplicas,
			Endpoint:      endpoint,
		})
	} else {
		ps := &inf.Status.PredictorStatuses[index]
		ps.Name = predictor.Name
		ps.Replicas = deploy.Status.Replicas
		ps.ReadyReplicas = deploy.Status.ReadyReplicas
		ps.Endpoint = endpoint
	}

	// Trim extra predictors status who have been removed from spec.
	if len(inf.Status.PredictorStatuses) > index+1 {
		inf.Status.PredictorStatuses = inf.Status.PredictorStatuses[:index+1]
	}
	return ctrl.Result{}, nil
}

// syncTrafficDistribution syncs traffic-routing resources to ensure traffic is distributed by the
// user configuration required.
// For example, user A serves two models and distribute 90% traffic to model A.1 and the left to
// model A.2, the network topology looks like this:
//
// User
//
//	|---request---> VirtualService
//	                      |--- 90% ---> Deploy-Of-Model-A.1
//	                      |--- 10% ---> Deploy-Of-Model-B.1
func (ir *InferenceReconciler) syncTrafficDistribution(inf *servingv1alpha1.Inference) error {
	vsvcInCluster := v1beta1.VirtualService{}
	vsvcExists := true
	err := ir.client.Get(context.Background(), types.NamespacedName{Namespace: inf.Namespace, Name: inf.Name}, &vsvcInCluster)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err != nil && errors.IsNotFound(err) {
		vsvcExists = false
	}

	// At this point, virtual service not found and create a new one.

	ratios := computePredictorTrafficRatios(inf)
	// Traffic will firstly hit virtual-service and then be dispatched to different
	// destinated model servers.
	vsvc := v1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inf.Name,
			Namespace: inf.Namespace,
		},
		Spec: beta1.VirtualService{
			Hosts:    []string{fmt.Sprintf("%s.*", inf.Name)},
			Gateways: []string{defaultIstioGatewayName},
		},
	}

	for pi := range inf.Spec.Predictors {
		predictor := &inf.Spec.Predictors[pi]
		vsvc.Spec.Http = append(vsvc.Spec.Http, &beta1.HTTPRoute{
			Name: predictor.Name,
			Route: []*beta1.HTTPRouteDestination{
				{
					Destination: &beta1.Destination{Host: svcHostForPredictor(inf, predictor)},
					Weight:      ratios[predictor.Name],
				},
			},
		})
	}

	// Sync virtual services for the first time, create it.
	if !vsvcExists {
		return ir.client.Create(context.Background(), &vsvc)
	}

	// Compare generated virtual service whether differs from in-cluster one or not, if so
	// update it to latest configuration.
	if !reflect.DeepEqual(vsvcInCluster.Spec, vsvc.Spec) {
		vsvcInCluster.Spec = *vsvc.Spec.DeepCopy()
		return ir.client.Update(context.Background(), &vsvcInCluster)
	}

	// Update latest traffic distribution of each predictor status.
	for psi := range inf.Status.PredictorStatuses {
		predictorStatus := &inf.Status.PredictorStatuses[psi]
		currentPercentage := ratios[predictorStatus.Name]
		predictorStatus.TrafficPercent = &currentPercentage
	}
	return nil
}

// syncServiceForInference sync an entrypoint service for inference service, it provides user with
// a unified host to access inference service, thought the inference service may serves more than one
// model version at a time, the backward network topology will handle traffic routing.
func (ir *InferenceReconciler) syncServiceForInference(inf *servingv1alpha1.Inference) error {
	svcInCluster := corev1.Service{}
	err := ir.client.Get(context.Background(), types.NamespacedName{Namespace: inf.Namespace, Name: inf.Name}, &svcInCluster)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	svcExists := true
	if err != nil && errors.IsNotFound(err) {
		svcExists = false
	}

	infSelector := map[string]string{
		apiv1.LabelInferenceName: inf.Name,
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: inf.Namespace,
			Name:      inf.Name,
			Labels:    infSelector,
		},
		Spec: corev1.ServiceSpec{
			Selector: infSelector,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       servingv1alpha1.DefaultPredictorContainerHttpPort,
					TargetPort: intstr.IntOrString{IntVal: servingv1alpha1.DefaultPredictorContainerHttpPort},
				},
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       servingv1alpha1.DefaultPredictorContainerGrpcPort,
					TargetPort: intstr.IntOrString{IntVal: servingv1alpha1.DefaultPredictorContainerGrpcPort},
				},
			},
		},
	}

	if !svcExists {
		return ir.client.Create(context.Background(), &svc)
	}

	if !reflect.DeepEqual(svc.Spec, corev1.ServiceSpec{
		Selector: svcInCluster.Spec.Selector,
		Type:     svcInCluster.Spec.Type,
		Ports:    svcInCluster.Spec.Ports,
	}) {
		svcInCluster.Spec.Selector = svc.Spec.Selector
		svcInCluster.Spec.Type = svc.Spec.Type
		svcInCluster.Spec.Ports = svc.Spec.Ports
		return ir.client.Update(context.Background(), &svcInCluster)
	}

	inf.Status.InferenceEndpoint = svcHostForInference(inf)
	return nil
}

func computePredictorTrafficRatios(inf *servingv1alpha1.Inference) map[string]int32 {
	defaultTrafficPercentage := int32(100)
	total := int32(0)
	ratios := make(map[string]int32)
	for pi := range inf.Spec.Predictors {
		predictor := &inf.Spec.Predictors[pi]
		percentage := defaultTrafficPercentage
		if inf.Spec.Predictors[pi].TrafficWeight != nil {
			percentage = *predictor.TrafficWeight
		}
		total += percentage
		ratios[predictor.Name] = percentage
	}

	for name, percent := range ratios {
		ratios[name] = percent * 100 / total
	}
	return ratios
}
