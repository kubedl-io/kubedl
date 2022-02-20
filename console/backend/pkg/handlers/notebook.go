package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/kubedl/apis/notebook/v1alpha1"
	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	consoleutils "github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/alibaba/kubedl/pkg/storage/backends"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewNotebookHandler(objStorage backends.ObjectStorageBackend) *NotebookHandler {
	return &NotebookHandler{
		objectBackend: objStorage,
		client:        clientmgr.GetCtrlClient(),
	}
}

// NotebookHandler handles the request for notebook
type NotebookHandler struct {
	client        client.Client
	objectBackend backends.ObjectStorageBackend
}

func (nh *NotebookHandler) ListNotebooksFromBackend(query *backends.NotebookQuery) ([]model.NotebookInfo, error) {
	notebooks, err := nh.objectBackend.ListNotebooks(query)
	if err != nil {
		return nil, err
	}
	notebookInfos := make([]model.NotebookInfo, 0, len(notebooks))
	for _, notebook := range notebooks {
		notebookInfos = append(notebookInfos, model.ConvertDMONotebookToNotebookInfo(notebook))
	}
	return notebookInfos, nil
}

func (nh *NotebookHandler) SubmitNotebook(data []byte) error {

	notebook := &v1alpha1.Notebook{}
	err := json.Unmarshal(data, notebook)

	klog.Infof("notebook notebook %v ", notebook)
	if err == nil {
		return nh.client.Create(context.Background(), notebook)
	} else {
		// fallback to yaml format
		err = yaml.Unmarshal(data, notebook)
		if err != nil {
			klog.Errorf("failed to unmarshal notebook in yaml format, data: %s", string(data))
			return err
		}
		return nh.client.Create(context.Background(), notebook)
	}
}

func (nh *NotebookHandler) DeleteNotebookFromBackend(ns, name, notebookID, region string) error {
	return nh.objectBackend.DeleteNotebook(ns, name, notebookID, region)
}

func (nh *NotebookHandler) GetYamlData(ns, name, kind string) ([]byte, error) {
	job := consoleutils.InitJobRuntimeObjectByKind(kind)
	if job == nil {
		return nil, fmt.Errorf("unsupported job kind: %s", kind)
	}

	err := nh.client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, job)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(job)
}

func (nh *NotebookHandler) GetJsonData(ns, name, kind string) ([]byte, error) {
	job := consoleutils.InitJobRuntimeObjectByKind(kind)
	if job == nil {
		return nil, fmt.Errorf("unsupported job kind: %s", kind)
	}

	err := nh.client.Get(context.Background(), types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, job)

	if err != nil {
		return nil, err
	}

	return json.Marshal(job)
}
