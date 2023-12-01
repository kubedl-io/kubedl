package api

import (
	"context"
	"encoding/json"
	"reflect"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"

	backendutils "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	tb "github.com/alibaba/kubedl/pkg/tensorboard"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTensorBoardController() *tensorBoardAPIsController {
	return &tensorBoardAPIsController{
		client:            backendutils.GetCtrlClient(),
		tensorboadHandler: handlers.NewTensorBoardHandler(),
	}
}

type tensorBoardAPIsController struct {
	client            client.Client
	tensorboadHandler *handlers.TensorBoardHandler
}

func (tc *tensorBoardAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	tbAPIs := routes.Group("/tensorboard")
	tbAPIs.GET("/status", tc.GetTensorBoardStatus)
	tbAPIs.POST("/reapply", tc.ReapplyTensorBoardInstance)
}

func (tc *tensorBoardAPIsController) GetTensorBoardStatus(c *gin.Context) {
	jobNamespace := c.Query("job_namespace")
	jobName := c.Query("job_name")
	jobUID := c.Query("job_uid")
	kind := c.Query("kind")

	klog.Infof("get tensorboard status of %s/%s/%s", jobNamespace, jobName, jobUID)

	job, err := tc.tensorboadHandler.GetJob(jobNamespace, jobName, kind)
	if err != nil {
		Error(c, err.Error())
		return
	}
	tbCfgBytes, ok := job.GetAnnotations()[apiv1.AnnotationTensorBoardConfig]
	if !ok {
		utils.Succeed(c, "no tensorboard configured")
		return
	}
	tbCfg := tb.TensorBoard{}
	_ = json.Unmarshal([]byte(tbCfgBytes), &tbCfg)

	tensorboard, err := tc.tensorboadHandler.GetTensorBoardInstance(jobNamespace, jobName, jobUID)
	if err != nil {
		Error(c, err.Error())
		return
	}
	if tensorboard == nil {
		utils.Succeed(c, tensorboardStatus{
			Name:      jobName,
			Namespace: jobNamespace,
			Config:    tbCfg,
			Phase:     "WaitForCreate",
			Message:   "Job has been created but tensorboard instance is waiting for creation",
			Ingresses: nil,
		})
		return
	}

	validIngresses, err := tc.tensorboadHandler.ListIngressInstances(jobNamespace, jobName, jobUID)
	if err != nil {
		Error(c, err.Error())
		return
	}

	utils.Succeed(c, wrapTensorBoardStatus(&tbCfg, tensorboard, validIngresses))
}

func (tc *tensorBoardAPIsController) ReapplyTensorBoardInstance(c *gin.Context) {
	jobNamespace := c.Query("job_namespace")
	jobName := c.Query("job_name")
	jobUID := c.Query("job_uid")
	kind := c.Query("kind")

	metaObj, err := tc.tensorboadHandler.GetJob(jobNamespace, jobName, kind)
	if err != nil {
		Error(c, err.Error())
		return
	}
	curCfgData, ok := metaObj.GetAnnotations()[apiv1.AnnotationTensorBoardConfig]
	curCfg := tb.TensorBoard{}
	if ok {
		if err = json.Unmarshal([]byte(curCfgData), &curCfg); err != nil {
			Error(c, "failed to unmarshal current tensorboard config, err: "+err.Error())
			return
		}
	}

	newCfgData, err := c.GetRawData()
	if err != nil {
		Error(c, err.Error())
		return
	}
	newCfg := tb.TensorBoard{}
	err = json.Unmarshal(newCfgData, &newCfg)
	if err != nil {
		Error(c, err.Error())
		return
	}

	dirtyCfg := !reflect.DeepEqual(curCfg, newCfg)

	tensorboard, err := tc.tensorboadHandler.GetTensorBoardInstance(jobNamespace, jobName, jobUID)
	if err != nil {
		Error(c, err.Error())
		return
	}
	if tensorboard != nil {
		_ = tc.client.Delete(context.Background(), tensorboard)
	}
	ingresses, err := tc.tensorboadHandler.ListIngressInstances(jobNamespace, jobName, jobUID)
	if err != nil {
		Error(c, err.Error())
		return
	}
	for _, ingress := range ingresses {
		_ = tc.client.Delete(context.Background(), ingress)
	}

	if dirtyCfg || tensorboard != nil {
		newCfg.UpdateTimestamp = metav1.Now()
		err = tc.tensorboadHandler.ApplyNewTensorBoardConfig(jobNamespace, jobName, kind, newCfg)
		if err != nil {
			Error(c, err.Error())
			return
		}
		utils.Succeed(c, "succeed to re-apply new tensorboard config")
		return
	}
	utils.Succeed(c, "tensorboard config not changed")
}

type simpleIngress struct {
	Name   string                     `json:"name"`
	Spec   networkingv1.IngressSpec   `json:"spec"`
	Status networkingv1.IngressStatus `json:"status"`
}

type tensorboardStatus struct {
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Config    tb.TensorBoard  `json:"config"`
	Phase     string          `json:"phase"`
	Message   string          `json:"message"`
	Ingresses []simpleIngress `json:"ingresses"`
}

func wrapTensorBoardStatus(config *tb.TensorBoard, pod *v1.Pod, ingresses []*networkingv1.Ingress) tensorboardStatus {
	status := tensorboardStatus{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Config:    *config,
		Phase:     string(pod.Status.Phase),
		Message:   pod.Status.Message,
	}

	for _, ingress := range ingresses {
		status.Ingresses = append(status.Ingresses, simpleIngress{
			Name:   ingress.Name,
			Spec:   ingress.Spec,
			Status: ingress.Status,
		})
	}

	return status
}
