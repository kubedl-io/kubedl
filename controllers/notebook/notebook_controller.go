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

package controllers

import (
	"context"
	"fmt"
	"path"

	"github.com/go-logr/logr"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/kubedl/cmd/options"

	notebookv1alpha1 "github.com/alibaba/kubedl/apis/notebook/v1alpha1"
)

func NewNotebookController(mgr ctrl.Manager, _ options.JobControllerConfiguration) *NotebookReconciler {
	r := &NotebookReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Notebook"),
		Scheme: mgr.GetScheme(),
	}
	r.recorder = mgr.GetEventRecorderFor("NotebookController")
	return r
}

// NotebookReconciler reconciles a Notebook object
type NotebookReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=notebook.kubedl.io,resources=notebooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=notebook.kubedl.io,resources=notebooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *NotebookReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("notebook", req.NamespacedName)

	notebook := &notebookv1alpha1.Notebook{}
	err := r.Get(context.Background(), req.NamespacedName, notebook)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("notebook doesn't exist", "name", req.String())
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "fail to get notebook", "name", req.String())
		return reconcile.Result{}, err
	}

	if notebook.Status.Condition == notebookv1alpha1.NotebookTerminated {
		r.Log.Info("notebook terminated", "name", req.String())
		return reconcile.Result{}, nil
	}

	// 1. create the notebook pod if not existing
	pod := &corev1.Pod{}
	nbPodName := types.NamespacedName{Namespace: notebook.Namespace, Name: nbNamePrefix(notebook.Name)}
	err = r.Get(context.Background(), nbPodName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create notebook pod " + nbPodName.String())
			pod.ObjectMeta = *notebook.Spec.Template.ObjectMeta.DeepCopy()
			pod.Name = nbPodName.Name
			pod.Namespace = nbPodName.Namespace
			if pod.Labels == nil {
				pod.Labels = make(map[string]string, 1)
			}
			pod.Labels[notebookv1alpha1.NotebookNameLabel] = notebook.Name
			pod.Spec = notebook.Spec.Template.Spec

			log.Infof("pod spec: %v", pod.Spec.Containers[0])
			err := setJuypterLabEnv(pod, notebook)
			if err != nil {
				return ctrl.Result{}, err
			}

			// add notebook as pod's owner reference, so that the pod is deleted when notebook is deleted
			addOwnerReferenceIfMissing(&pod.ObjectMeta, notebook)

			// run juypter as root so that user can be switch to root user if needed.
			// The container still run as jovyan user when the container launches.
			root := int64(0)
			pod.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  &root,
				RunAsGroup: &root,
			}

			// create the notebook pod
			err = r.Create(context.Background(), pod)
			if err != nil {
				r.Log.Error(err, "failed to create pod "+nbPodName.String())
				return reconcile.Result{Requeue: true}, err
			}
			r.recorder.Eventf(notebook, core.EventTypeNormal, "SuccessfulCreatedPod", "Created pod %v", pod.Name)
			notebook.Status.Condition = notebookv1alpha1.NotebookCreated
			notebook.Status.Message = "created notebook pod " + pod.Name
			notebook.Status.LastTransitionTime = metav1.Now()
			err = r.Status().Update(context.Background(), notebook)
			if err != nil {
				r.Log.Error(err, "failed to update notebook status", "notebook", notebook.Name)
				return ctrl.Result{}, err
			}
		} else {
			// for other errors
			return reconcile.Result{}, err
		}
	}

	// check pod status and update notebook status
	shouldReconcile, err := r.updateNotebookCondition(pod, notebook)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 2. create the notebook service if not existing
	nbServiceName := getNotebookServiceName(notebook)
	nbService := &corev1.Service{}
	err = r.Get(context.Background(), nbServiceName, nbService)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create notebook service " + nbServiceName.String())
			nbService = generateService(pod, notebook)

			// create the notebook service
			err = r.Create(context.Background(), nbService)
			if err != nil {
				r.Log.Error(err, "failed to create service "+nbServiceName.String())
				return reconcile.Result{Requeue: true}, err
			}
			r.recorder.Eventf(notebook, core.EventTypeNormal, "SuccessfulCreatedService", "Created service %v", nbService.Name)
		} else {
			// for other errors
			return reconcile.Result{}, err
		}
	}

	// 3. create the ingress if not existing
	nbIngressName := types.NamespacedName{Namespace: notebook.Namespace, Name: nbNamePrefix(notebook.Name)}
	nbIngress := &v1.Ingress{}
	err = r.Get(context.Background(), nbIngressName, nbIngress)
	ingressPath := getIngressPath(nbService)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("create notebook ingress " + nbIngressName.String())
			ingress := generateIngress(nbService, notebook, ingressPath)
			// create the notebook ingress
			err = r.Create(context.Background(), ingress)
			if err != nil {
				r.Log.Error(err, "failed to create ingress "+nbIngressName.String())
				return reconcile.Result{Requeue: true}, err
			}
			r.recorder.Eventf(notebook, core.EventTypeNormal, "SuccessfulCreateIngress", "Created ingress %v", nbService.Name)
		} else {
			// for other errors
			return reconcile.Result{}, err
		}
	}

	if notebook.Status.Url == "" {
		if len(nbIngress.Status.LoadBalancer.Ingress) > 0 {
			fullUrl := "http://"
			ingress := nbIngress.Status.LoadBalancer.Ingress[0]
			if ingress.IP != "" {
				fullUrl += ingress.IP + ingressPath + "/"
			} else if ingress.Hostname != "" {
				fullUrl += ingress.Hostname + ingressPath + "/"
			}
			notebook.Status.Url = fullUrl
			// update notebook url
			err = r.Status().Update(context.Background(), notebook)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			shouldReconcile = true
		}
	}
	return ctrl.Result{Requeue: shouldReconcile}, nil
}

func getNotebookServiceName(notebook *notebookv1alpha1.Notebook) types.NamespacedName {
	nbServiceName := types.NamespacedName{Namespace: notebook.Namespace, Name: nbNamePrefix(notebook.Name)}
	return nbServiceName
}

func (r *NotebookReconciler) updateNotebookCondition(pod *corev1.Pod, notebook *notebookv1alpha1.Notebook) (bool, error) {
	var err error
	var shouldReconcile bool

	switch pod.Status.Phase {
	case corev1.PodPending:
		if notebook.Status.Condition == "" {
			notebook.Status.Condition = notebookv1alpha1.NotebookCreated
			notebook.Status.Message = "created notebook pod " + pod.Name
			notebook.Status.LastTransitionTime = metav1.Now()
			err = r.Status().Update(context.Background(), notebook)
		}
		shouldReconcile = true
	case corev1.PodRunning:
		if notebook.Status.Condition != notebookv1alpha1.NotebookRunning {
			notebook.Status.Condition = notebookv1alpha1.NotebookRunning
			notebook.Status.LastTransitionTime = metav1.Now()
			notebook.Status.Message = "notebook pod " + pod.Name + " is running"
			err = r.Status().Update(context.Background(), notebook)
		}
	case corev1.PodFailed, corev1.PodSucceeded:
		if notebook.Status.Condition != notebookv1alpha1.NotebookTerminated {
			notebook.Status.Condition = notebookv1alpha1.NotebookTerminated
			notebook.Status.LastTransitionTime = metav1.Now()
			notebook.Status.Message = "notebook pod " + pod.Name + " terminated: " + string(pod.Status.Phase)
			err = r.Status().Update(context.Background(), notebook)
		}
	}
	if err != nil {
		r.Log.Error(err, "failed to update notebook status", "notebook", notebook.Name)
		return true, err
	}
	return shouldReconcile, err
}

func generateIngress(svc *corev1.Service, nb *notebookv1alpha1.Notebook, ingressPath string) *v1.Ingress {
	prefixPathType := v1.PathTypePrefix

	// http://{host}/notebooks/{namespace}/{svc.Name}/
	// all traffic requests for this url will be forwarded to the backend service and its endpoint.
	ingress := &v1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nbNamePrefix(nb.Name),
			Namespace: nb.Namespace,
			Labels:    svc.Labels,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     ingressPath,
									PathType: &prefixPathType,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: svc.Name,
											Port: v1.ServiceBackendPort{
												Name: notebookv1alpha1.NotebookPortName,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	addOwnerReferenceIfMissing(&ingress.ObjectMeta, nb)
	return ingress
}

func getIngressPath(svc *corev1.Service) string {
	nbPath := path.Join("/", "notebooks", svc.Namespace, svc.Name)
	return nbPath
}

func nbNamePrefix(nbName string) string {
	return "nb-" + nbName
}

// func addNotebookDefaultPort(spec *corev1.PodSpec) {
// 	index := 0
// 	for i, container := range spec.Containers {
// 		if container.Name == notebookv1alpha1.NotebookContainerName {
// 			index = i
// 			break
// 		}
// 	}
//
// 	hasNotebookPort := false
// 	for _, port := range spec.Containers[index].Ports {
// 		if port.Name == notebookv1alpha1.NotebookPortName {
// 			hasNotebookPort = true
// 			break
// 		}
// 	}
// 	if !hasNotebookPort {
// 		spec.Containers[index].Ports = append(spec.Containers[index].Ports, corev1.ContainerPort{
// 			Name:          notebookv1alpha1.NotebookPortName,
// 			ContainerPort: notebookv1alpha1.NotebookDefaultPort,
// 		})
// 	}
// }

// set necessary ENVs in notebook container's env
func setJuypterLabEnv(pod *corev1.Pod, notebook *notebookv1alpha1.Notebook) error {
	if len(pod.Spec.Containers) == 0 {
		err := fmt.Errorf("pod %s container list is zero", pod.Name)
		return err
	}
	containers := pod.Spec.Containers
	nbContainerExists := false
	for i, container := range containers {
		if container.Name == notebookv1alpha1.NotebookContainerName {

			juypterLabEnvExists := false
			notebookArgsExists := false
			grantSudoExists := false
			juypterAllowInsecureWritesExists := false

			for _, env := range container.Env {
				if env.Name == "JUPYTER_ENABLE_LAB" {
					juypterLabEnvExists = true
				}
				if env.Name == "NOTEBOOK_ARGS" {
					notebookArgsExists = true
					nbEnv := generateNotebookArgsEnv(notebook)
					container.Env[i].Value = nbEnv + " " + container.Env[i].Value
				}
				if env.Name == "GRANT_SUDO" {
					grantSudoExists = true
				}
				if env.Name == "JUPYTER_ALLOW_INSECURE_WRITES" {
					juypterAllowInsecureWritesExists = true
				}
			}
			if !juypterLabEnvExists {
				containers[i].Env = append(containers[i].Env, corev1.EnvVar{
					Name:  "JUPYTER_ENABLE_LAB",
					Value: "yes",
				})
			}

			if !notebookArgsExists {
				containers[i].Env = append(containers[i].Env, corev1.EnvVar{
					Name:  "NOTEBOOK_ARGS",
					Value: generateNotebookArgsEnv(notebook),
				})
			}

			if !grantSudoExists {
				containers[i].Env = append(containers[i].Env, corev1.EnvVar{
					Name:  "GRANT_SUDO",
					Value: "yes",
				})
			}

			if !juypterAllowInsecureWritesExists {
				containers[i].Env = append(containers[i].Env, corev1.EnvVar{
					Name:  "JUPYTER_ALLOW_INSECURE_WRITES",
					Value: "true",
				})
			}

			nbContainerExists = true
			break
		}
	}
	if !nbContainerExists {
		err := fmt.Errorf("pod %s doesn't have container named 'notebook', change the notebook container name to 'notebook'", pod.Name)
		return err
	}
	return nil
}

// change the base url and pass in noToken
func generateNotebookArgsEnv(notebook *notebookv1alpha1.Notebook) string {
	nbServiceName := getNotebookServiceName(notebook)
	ingressPath := path.Join("/", "notebooks", nbServiceName.Namespace, nbServiceName.Name)
	baseUrl := fmt.Sprintf("--LabApp.base_url=%s --NotebookApp.base_url=%s --ServerApp.base_url=%s --ServerApp.allow_root=true", ingressPath, ingressPath, ingressPath)
	noToken := "--LabApp.token=''"
	return baseUrl + " " + noToken
}

// findNotebookPort finds the notebook port from spec
func findNotebookPort(pod *corev1.Pod) int32 {
	containers := pod.Spec.Containers
	for _, container := range containers {
		if container.Name == notebookv1alpha1.NotebookContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == notebookv1alpha1.NotebookPortName {
					return port.ContainerPort
				}
			}
		}
	}
	return notebookv1alpha1.NotebookDefaultPort
}

func generateService(pod *corev1.Pod, nb *notebookv1alpha1.Notebook) *corev1.Service {
	serviceName := nbNamePrefix(nb.Name)
	labels := map[string]string{
		notebookv1alpha1.NotebookNameLabel: nb.Name,
	}
	port := findNotebookPort(pod)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: nb.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name:       notebookv1alpha1.NotebookPortName,
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
				},
			},
		},
	}

	addOwnerReferenceIfMissing(&service.ObjectMeta, nb)
	return service
}

// addOwnerReferenceIfMissing adds notebook to pod's OwnerReferences
func addOwnerReferenceIfMissing(meta *metav1.ObjectMeta, notebook *notebookv1alpha1.Notebook) {
	if meta.OwnerReferences == nil {
		meta.OwnerReferences = make([]metav1.OwnerReference, 0)
	}
	exists := false
	for _, ref := range meta.OwnerReferences {
		if ref.Kind == "Notebook" && ref.UID == notebook.UID {
			// already exists
			exists = true
			break
		}
	}
	if !exists {
		meta.OwnerReferences = append(meta.OwnerReferences, metav1.OwnerReference{
			APIVersion:         notebook.APIVersion,
			Kind:               notebook.Kind,
			Name:               notebook.Name,
			UID:                notebook.UID,
			BlockOwnerDeletion: pointer.BoolPtr(true),
			Controller:         pointer.BoolPtr(true),
		})
	}
}

func (r *NotebookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&notebookv1alpha1.Notebook{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&v1.Ingress{}).
		Complete(r)
}
