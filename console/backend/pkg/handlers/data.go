package handlers

import (
	"context"
	"sort"

	"github.com/alibaba/kubedl/pkg/util/resource_utils"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"

	corev1 "k8s.io/api/core/v1"
	resources "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
)

const (
	ResourceGPU   = "nvidia.com/gpu"
	IndexNodeName = "spec.nodeName"
	IndexPhase    = "status.phase"
	GPUType       = "aliyun.accelerator/nvidia_name"
)

func NewDataHandler() *DataHandler {
	_ = clientmgr.IndexField(&corev1.Pod{}, IndexNodeName, func(obj client.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return []string{}
		}
		if len(pod.Spec.NodeName) == 0 {
			return []string{}
		}
		return []string{pod.Spec.NodeName}
	})

	_ = clientmgr.IndexField(&corev1.Pod{}, IndexPhase, func(obj client.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return []string{}
		}
		return []string{string(pod.Status.Phase)}
	})

	return &DataHandler{client: clientmgr.GetCtrlClient()}
}

type DataHandler struct {
	client client.Client
}

// GetClusterTotalResource sum all nodes allocatable resource(cpu/memory/gpu)
func (ov *DataHandler) GetClusterTotalResource() (model.ClusterTotalResource, error) {
	nodes := &corev1.NodeList{}
	err := ov.client.List(context.TODO(), nodes)
	if err != nil {
		klog.Errorf("GetClusterTotalResource Failed to list nodes")
		return model.ClusterTotalResource{}, err
	}
	totalResource := corev1.ResourceList{}
	for _, node := range nodes.Items {
		allocatable := node.Status.Allocatable.DeepCopy()
		totalResource = resources.Add(totalResource, allocatable)
	}
	clusterTotal := model.ClusterTotalResource{
		TotalCPU:    totalResource.Cpu().MilliValue(),
		TotalMemory: totalResource.Memory().Value(),
		TotalGPU:    getGpu(totalResource).MilliValue()}
	return clusterTotal, nil
}

// GetClusterRequestResource sum all pods request resource(cpu/memory/gpu)
func (ov *DataHandler) GetClusterRequestResource(podPhase string) (model.ClusterRequestResource, error) {
	namespaces := &corev1.NamespaceList{}
	err := ov.client.List(context.TODO(), namespaces)
	if err != nil {
		klog.Errorf("GetClusterRequestResource Failed to list namespaces")
		return model.ClusterRequestResource{}, err
	}
	totalRequest := corev1.ResourceList{}
	for _, namespace := range namespaces.Items {
		// query pod list in namespace
		podList := &corev1.PodList{}
		err = ov.client.List(context.TODO(), podList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(IndexPhase, podPhase),
			Namespace:     namespace.Name})
		if err != nil {
			klog.Errorf("GetClusterRequestResource Failed to get pod list on node: %v error: %v", namespace.Name, err)
			return model.ClusterRequestResource{}, err
		}
		totalRequest = resources.Add(totalRequest, getPodRequest(podList, corev1.PodPhase(podPhase)))
	}
	clusterRequest := model.ClusterRequestResource{
		RequestCPU:    totalRequest.Cpu().MilliValue(),
		RequestMemory: totalRequest.Memory().Value(),
		RequestGPU:    getGpu(totalRequest).MilliValue()}
	return clusterRequest, nil
}

// GetClusterNodeInfos sum all pods allocatable and request resource(cpu/memory/gpu) in a node
func (ov *DataHandler) GetClusterNodeInfos() (model.ClusterNodeInfoList, error) {
	nodes := &corev1.NodeList{}
	err := ov.client.List(context.TODO(), nodes)
	if err != nil {
		klog.Errorf("GetClusterTotalResource Failed to list nodes")
		return model.ClusterNodeInfoList{}, err
	}
	var clusterNodeInfos []model.ClusterNodeInfo
	for _, node := range nodes.Items {
		// query pod list in node
		podList := &corev1.PodList{}
		err = ov.client.List(context.TODO(), podList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(IndexNodeName, node.Name),
		})
		if err != nil {
			klog.Errorf("GetClusterNodeInfos Failed to get pod list on node: %v error: %v", node.Name, err)
			return model.ClusterNodeInfoList{}, err
		}
		podsRequest := getPodRequest(podList, corev1.PodRunning)
		allocatable := node.Status.Allocatable.DeepCopy()

		clusterNodeInfo := model.ClusterNodeInfo{
			NodeName:      node.Name,
			InstanceType:  getInstanceType(&node),
			GPUType:       node.Labels[GPUType],
			TotalCPU:      allocatable.Cpu().MilliValue(),
			TotalMemory:   allocatable.Memory().Value(),
			TotalGPU:      getGpu(allocatable).MilliValue(),
			RequestCPU:    podsRequest.Cpu().MilliValue(),
			RequestMemory: podsRequest.Memory().Value(),
			RequestGPU:    getGpu(podsRequest).MilliValue(),
		}
		clusterNodeInfos = append(clusterNodeInfos, clusterNodeInfo)
	}

	// sort by node name
	sort.SliceStable(clusterNodeInfos, func(i, j int) bool {
		return clusterNodeInfos[i].NodeName > clusterNodeInfos[j].NodeName
	})

	return model.ClusterNodeInfoList{Items: clusterNodeInfos}, nil
}

// sum podlist request
func getPodRequest(podList *corev1.PodList, phase corev1.PodPhase) corev1.ResourceList {
	totalRequest := corev1.ResourceList{}
	for _, pod := range podList.Items {
		if pod.Status.Phase != phase {
			continue
		}
		totalRequest = resources.Add(totalRequest, resource_utils.ComputePodResourceRequest(&pod))
	}
	return totalRequest
}

// get node instance type ,get from labels compatible
func getInstanceType(node *corev1.Node) string {
	instanceType := node.Labels["node.kubernetes.io/instance-type"]
	if instanceType == "" {
		instanceType = node.Labels["beta.kubernetes.io/instance-type"]
	}
	return instanceType
}

// get gpu from custom resource map
func getGpu(resourceList corev1.ResourceList) *resource.Quantity {
	if val, ok := (resourceList)[ResourceGPU]; ok {
		return &val
	}
	return &resource.Quantity{Format: resource.DecimalSI}
}
