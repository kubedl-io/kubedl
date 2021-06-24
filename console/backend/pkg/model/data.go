package model

type ClusterTotalResource struct {
	TotalCPU    int64 `json:"totalCPU"`
	TotalMemory int64 `json:"totalMemory"`
	TotalGPU    int64 `json:"totalGPU"`
}

type ClusterRequestResource struct {
	RequestCPU    int64 `json:"requestCPU"`
	RequestMemory int64 `json:"requestMemory"`
	RequestGPU    int64 `json:"requestGPU"`
}

type ClusterNodeInfo struct {
	NodeName      string `json:"nodeName"`
	InstanceType  string `json:"instanceType"`
	GPUType       string `json:"gpuType"`
	TotalCPU      int64  `json:"totalCPU"`
	TotalMemory   int64  `json:"totalMemory"`
	TotalGPU      int64  `json:"totalGPU"`
	RequestCPU    int64  `json:"requestCPU"`
	RequestMemory int64  `json:"requestMemory"`
	RequestGPU    int64  `json:"requestGPU"`
}

type ClusterNodeInfoList struct {
	Items []ClusterNodeInfo `json:"items,omitempty"`
}
