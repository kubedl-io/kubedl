package tensorboard

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/kubedl/pkg/job_controller"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	commonutil "github.com/alibaba/kubedl/pkg/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const port = 6006

var tbRT = strings.ToLower(string(apiv1.ReplicaTypeTensorBoard))

// TensorBoard defines the desired state of tensorboard
type TensorBoard struct {
	// LogDir is the location of the log directory to be read by tensorboard.
	LogDir string `json:"logDir,omitempty"`
	// TTLSecondsAfterJobFinished is the TTL to clean up the tensorboard after the job is finished.
	TTLSecondsAfterJobFinished int32 `json:"ttlSecondsAfterJobFinished,omitempty"`
	// UpdateTimestamp is the timestamp of the last update
	UpdateTimestamp metav1.Time `json:"updateTimestamp,omitempty"`
	// Image is tensorboard container image.
	Image *string `json:"image,omitempty"`
	// Ingress defines the desired state of ingress
	Ingress *IngressSpec `json:"ingressSpec,omitempty"`
}

// IngressSpec defines the desired state of ingress
type IngressSpec struct {
	// Host is only checked if the Type is Ingress.
	Host *string `json:"host,omitempty"`
	// PathPrefix is only checked if the Type is Ingress.
	PathPrefix *string `json:"pathPrefix,omitempty"`
	// Annotations is an unstructured key value map stored for ingress.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ReconcileTensorBoard reads that state of the cluster for the TensorBoard object and makes changes based on the state read.
func ReconcileTensorBoard(jc job_controller.JobController, c client.Client, object client.Object, masterType apiv1.ReplicaType, spec *apiv1.ReplicaSpec,
	jobStatus apiv1.JobStatus, result *reconcile.Result) error {

	opts := TensorBoard{}
	cfg, ok := object.GetAnnotations()[apiv1.AnnotationTensorBoardConfig]
	if ok {
		if err := json.Unmarshal([]byte(cfg), &opts); err != nil {
			return err
		}
	} else {
		// annotation not found ,try to delete tensorboard
		if err := deleteTensorBoard(jc, c, object); err != nil {
			return err
		}
		return nil
	}

	if needSync, err := cleanupAndCheckNeedSync(jc, c, object, jobStatus, result, opts); err != nil {
		return err
	} else if !needSync {
		log.Infof("no need to sync tensorboard, job: %s/%s", object.GetNamespace(), object.GetName())
		return nil
	}

	if err := syncPod(jc, c, object, masterType, spec, opts, cfg); err != nil {
		log.Errorf("TensorBoard sync pod error %v", err)
		return err
	}

	if err := syncService(jc, c, object, opts); err != nil {
		log.Errorf("TensorBoard sync service error %v", err)
		return err
	}

	if err := syncIngress(jc, c, object, opts, cfg); err != nil {
		log.Errorf("TensorBoard sync ingress error %v", err)
		return err
	}

	return nil
}

// cleanupAndCheckNeedSync cleanup the tensorboard if needed and check if the tensorboard need sync
func cleanupAndCheckNeedSync(jc job_controller.JobController, c client.Client, object client.Object,
	jobStatus apiv1.JobStatus, result *reconcile.Result, opts TensorBoard) (bool, error) {

	if commonutil.IsSucceeded(jobStatus) || commonutil.IsFailed(jobStatus) {
		currentTime := time.Now()
		if jobStatus.CompletionTime == nil {
			return false, fmt.Errorf("cleanup TensorBoard for Job %s, but job has CompletionTime not set", object.GetName())
		}
		duration := time.Second * time.Duration(opts.TTLSecondsAfterJobFinished)
		deleteTime := jobStatus.CompletionTime.Add(duration)
		// if the creation time of the tensorboard config is later than
		// the comletion time of the job, the deletion time of the tensorboard
		// needs to be calculated from the creation time of the tensorboard config.
		if jobStatus.CompletionTime.Before(&opts.UpdateTimestamp) {
			deleteTime = opts.UpdateTimestamp.Add(duration)
		}
		if currentTime.After(deleteTime) {
			log.Infof("delete tensorboard, TTL: %d", opts.TTLSecondsAfterJobFinished)
			// Remove tensorboard config from annotation and clean up tensorboard instances/ingress
			// in next reconcile.
			anno := object.GetAnnotations()
			delete(anno, apiv1.AnnotationTensorBoardConfig)
			object.SetAnnotations(anno)
			if err := c.Update(context.Background(), object); err != nil {
				return true, err
			}
			return false, nil
		} else {
			after := deleteTime.Sub(currentTime)
			result.Requeue = true
			if result.RequeueAfter > after || result.RequeueAfter == 0 {
				result.RequeueAfter = after
			}
			log.Infof("reconcile after %v, job: %s/%s", after, object.GetNamespace(), object.GetName())
		}
	}
	return true, nil
}

// syncService sync the tensorboard's pod
func syncPod(jc job_controller.JobController, c client.Client, metaObj metav1.Object, masterType apiv1.ReplicaType,
	spec *apiv1.ReplicaSpec, opts TensorBoard, cfg string) error {

	// get pod from cache
	pod := &corev1.Pod{}
	index := strconv.Itoa(0)
	name := commonutil.GenGeneralName(metaObj.GetName(), tbRT, index)
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: metaObj.GetNamespace(),
			Name:      name},
		pod); err == nil {
		if !metav1.IsControlledBy(pod, metaObj) {
			return fmt.Errorf("sync TensorBoard %v pod conflict", metaObj.GetName())
		}

		// check if the config is changed
		oldOpts := TensorBoard{}
		oldCfg, ok := pod.GetAnnotations()[apiv1.AnnotationTensorBoardConfig]
		if ok {
			if err := json.Unmarshal([]byte(oldCfg), &oldOpts); err != nil {
				return err
			}
		}
		oldOpts.UpdateTimestamp = opts.UpdateTimestamp
		if reflect.DeepEqual(oldOpts, opts) {
			return nil

		}
		// the config is changed, delete the pod and create a new one.
		if err := c.Delete(context.Background(), pod); err != nil {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	log.Infof("TensorBoard %v sync pod not found.", name)

	// pod not found
	template := spec.Template.DeepCopy()
	template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	labels := map[string]string{}
	labels[apiv1.ReplicaTypeLabel] = tbRT
	labels[apiv1.ReplicaIndexLabel] = index
	labels[apiv1.ReplicaNameLabel] = name
	labels[apiv1.JobNameLabel] = metaObj.GetName()

	template.Labels = commonutil.MergeMap(template.Labels, labels)

	if template.Annotations == nil {
		template.Annotations = map[string]string{}
	}
	template.Annotations[apiv1.AnnotationTensorBoardConfig] = cfg

	pathPrefix := ""
	if opts.Ingress != nil && opts.Ingress.PathPrefix != nil {
		pathPrefix = *opts.Ingress.PathPrefix
	}

	for i := range template.Spec.Containers {
		c := &template.Spec.Containers[i]
		if c.Name == jc.Controller.GetDefaultContainerName() {
			c.Name = "tensorboard"
			// Remove the GPU device to prevent the pod from failing to start
			c.Env = setEnv(c.Env, "NVIDIA_VISIBLE_DEVICES", "")
			if opts.Image != nil && len(*opts.Image) > 0 {
				c.Image = *opts.Image
			}
			c.Command = []string{
				"/bin/sh",
				"-c",
				fmt.Sprintf("python -m tensorboard.main --logdir %s --path_prefix %s --host=`hostname`", opts.LogDir, path.Join("/", pathPrefix, metaObj.GetNamespace(), metaObj.GetName())),
			}
			c.Resources = corev1.ResourceRequirements{}

			c.Ports = []corev1.ContainerPort{
				{ContainerPort: port},
			}
		}
	}

	if err := jc.CreatePod(metaObj, tbRT, index, template, false); err != nil {
		return err
	}

	return nil
}

// syncService sync the tensorboard's service
func syncService(jc job_controller.JobController, c client.Client,
	metaObj metav1.Object, opts TensorBoard) error {

	// get service from cache
	service := &corev1.Service{}
	index := strconv.Itoa(0)
	name := commonutil.GenGeneralName(metaObj.GetName(), tbRT, index)
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: metaObj.GetNamespace(),
			Name:      name},
		service); err == nil {

		if !metav1.IsControlledBy(service, metaObj) {
			return fmt.Errorf("sync TensorBoard %v service conflict", metaObj.GetName())
		}
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	log.Infof("TensorBoard %v sync service not found.", name)

	// service not found
	labels := map[string]string{}
	labels[apiv1.ReplicaTypeLabel] = tbRT
	labels[apiv1.ReplicaIndexLabel] = index
	labels[apiv1.ReplicaNameLabel] = name

	service = &corev1.Service{
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name: "tensorboard",
					Port: port,
				},
			},
		},
	}
	service.Labels = commonutil.MergeMap(service.Labels, labels)

	if err := jc.CreateService(metaObj, apiv1.ReplicaTypeTensorBoard, service, index); err != nil {
		return err
	}

	return nil
}

// syncService sync the tensorboard's ingress
func syncIngress(jc job_controller.JobController, c client.Client,
	metaObj metav1.Object, opts TensorBoard, cfg string) error {

	ingSpec := opts.Ingress
	if ingSpec == nil {
		return nil
	}
	pathPrefix := ""
	if ingSpec.PathPrefix != nil {
		pathPrefix = *ingSpec.PathPrefix
	}

	// get ingress from cache
	in := &v1.Ingress{}
	index := strconv.Itoa(0)
	name := commonutil.GenGeneralName(metaObj.GetName(), tbRT, index)
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: metaObj.GetNamespace(),
			Name:      name},
		in); err == nil {
		if !metav1.IsControlledBy(in, metaObj) {
			return fmt.Errorf("sync TensorBoard %v ingress conflict", metaObj.GetName())
		}

		// check if the config is changed
		oldOpts := TensorBoard{}
		oldCfg, ok := in.GetAnnotations()[apiv1.AnnotationTensorBoardConfig]
		if ok {
			if err := json.Unmarshal([]byte(oldCfg), &oldOpts); err != nil {
				return err
			}
		}
		oldOpts.UpdateTimestamp = opts.UpdateTimestamp
		if reflect.DeepEqual(oldOpts, opts) {
			return nil

		}
		// the config is changed, delete the ingress and create a new one.
		if err := c.Delete(context.Background(), in); err != nil {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	// ingress not found
	labels := jc.GenLabels(metaObj.GetName())
	labels[apiv1.ReplicaTypeLabel] = tbRT
	labels[apiv1.ReplicaIndexLabel] = index
	prefixPathType := v1.PathTypePrefix

	in = &v1.Ingress{
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path:     path.Join("/", pathPrefix, metaObj.GetNamespace(), metaObj.GetName()) + "/",
									PathType: &prefixPathType,
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: name,
											Port: v1.ServiceBackendPort{
												Number: port,
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
	in.Namespace = metaObj.GetNamespace()
	in.Name = name
	in.Labels = labels

	if ingSpec.Annotations != nil {
		in.Annotations = ingSpec.Annotations
	}

	if in.Annotations == nil {
		in.Annotations = map[string]string{}
	}
	in.Annotations[apiv1.AnnotationTensorBoardConfig] = cfg
	in.Annotations["kubernetes.io/ingress.class"] = "nginx"

	// Create OwnerReference.
	controllerRef := jc.GenOwnerReference(metaObj)
	in.OwnerReferences = append(in.OwnerReferences, *controllerRef)

	if err := c.Create(context.Background(), in); err != nil {
		log.Errorf("fail to create ingress %s", name)
		return err
	}
	log.Infof("create ingress for tensorboard %v", name)

	return nil
}

// deleteTensorBoard delete tensorboard ingress service and pod.
func deleteTensorBoard(jc job_controller.JobController, c client.Client, metaObj metav1.Object) error {
	index := strconv.Itoa(0)
	ns := metaObj.GetNamespace()
	name := commonutil.GenGeneralName(metaObj.GetName(), tbRT, index)

	// delete ingress
	in := v1.Ingress{}
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: ns,
			Name:      name},
		&in); err == nil && metav1.IsControlledBy(&in, metaObj) {
		if err := c.Delete(context.Background(), &in); err != nil {
			return err
		}
		log.Infof("delete tensorboard %s/%s ingress success.", ns, name)
	} else if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// delete service
	service := corev1.Service{}
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: ns,
			Name:      name},
		&service); err == nil && metav1.IsControlledBy(&service, metaObj) {
		if err := c.Delete(context.Background(), &service); err != nil {
			return err
		}
		log.Infof("delete tensorboard %s/%s service success.", ns, name)
	} else if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// delete pod
	pod := corev1.Pod{}
	if err := c.Get(context.Background(),
		types.NamespacedName{
			Namespace: ns,
			Name:      name},
		&pod); err == nil && metav1.IsControlledBy(&pod, metaObj) {
		if err := c.Delete(context.Background(), &pod); err != nil {
			return err
		}
		log.Infof("delete tensorboard %s/%s pod success.", ns, name)
	} else if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func setEnv(env []corev1.EnvVar, key, value string) []corev1.EnvVar {
	if env == nil {
		env = []corev1.EnvVar{}
	}
	for i := range env {
		e := &env[i]
		if e.Name == key {
			e.Value = value
			return env
		}
	}
	return append(env, corev1.EnvVar{Name: key, Value: value})
}
