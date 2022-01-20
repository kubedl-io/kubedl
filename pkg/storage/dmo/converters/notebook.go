package converters

import (
	"encoding/json"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"
	"github.com/alibaba/kubedl/pkg/util/tenancy"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

func ConvertNotebookToDMONotebook(notebook *v1alpha1.Notebook, region string) (*dmo.Notebook, error) {
	klog.V(5).Infof("[ConvertNotebookToDMONotebook] notebook: %s/%s", notebook.Namespace, notebook.Name)
	dmoNotebook := dmo.Notebook{
		Name:       notebook.GetName(),
		Namespace:  notebook.GetNamespace(),
		NotebookID: string(notebook.GetUID()),
		Version:    notebook.GetResourceVersion(),
		Resources:  "",
		GmtCreated: notebook.GetCreationTimestamp().Time,
		Url:        notebook.Status.Url,
	}

	if region != "" {
		dmoNotebook.DeployRegion = &region
	}

	if tn, err := tenancy.GetTenancy(notebook); err == nil && tn != nil {
		dmoNotebook.Tenant = &tn.Tenant
		dmoNotebook.Owner = &tn.User
		if dmoNotebook.DeployRegion == nil && tn.Region != "" {
			dmoNotebook.DeployRegion = &tn.Region
		}
	} else {
		dmoNotebook.Tenant = pointer.StringPtr("")
		dmoNotebook.Owner = pointer.StringPtr("")
	}

	dmoNotebook.Status = string(notebook.Status.Condition)

	if dmoNotebook.Status == string(v1alpha1.NotebookRunning) {
		dmoNotebook.GmtRunning = &notebook.Status.LastTransitionTime.Time

	}

	if dmoNotebook.Status == string(v1alpha1.NotebookTerminated) {
		dmoNotebook.GmtTerminated = &notebook.Status.LastTransitionTime.Time
	}

	dmoNotebook.Deleted = util.IntPtr(0)
	dmoNotebook.IsInEtcd = util.IntPtr(1)

	resources := computePodResources(&notebook.Spec.Template.Spec)
	resourcesBytes, err := json.Marshal(&resources)
	if err != nil {
		return nil, err
	}

	dmoNotebook.Resources = string(resourcesBytes)
	return &dmoNotebook, nil
}
