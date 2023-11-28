package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
)

const (
	CodesourceConfigMapName = "kubedl-codesource-config"
	CodesourceConfigMapKey  = "codesource"
)

func NewCodeSourceHandler() *CodeSourceHandler {
	return &CodeSourceHandler{client: clientmgr.GetCtrlClient()}
}

type CodeSourceHandler struct {
	client client.Client
}

// post
func (ov *CodeSourceHandler) PostCodeSourceToConfigMap(codeSource model.CodeSource) error {
	klog.Infof("CodeSource : %s", codeSource)

	configMap, err := getOrCreateCodeSourceConfigMap()
	if err != nil {
		return err
	}

	codeSourceMap, err := getCodeSourceMap(configMap)
	if err != nil {
		return err
	}

	_, exists := codeSourceMap[codeSource.Name]
	if exists {
		klog.Errorf("CodeSource exists, name: %s", codeSource.Name)
		return fmt.Errorf("CodeSource exists, name: %s", codeSource.Name)
	}

	codeSourceMap[codeSource.Name] = codeSource

	return setCodeSourceConfigMap(configMap, codeSourceMap)
}

// delete
func (ov *CodeSourceHandler) DeleteCodeSourceFromConfigMap(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is empty")
	}

	configMap, err := getOrCreateCodeSourceConfigMap()
	if err != nil {
		return err
	}

	codeSourceMap, err := getCodeSourceMap(configMap)
	if err != nil {
		return err
	}

	_, exists := codeSourceMap[name]
	if !exists {
		klog.Errorf("CodeSource not exists, name: %s", name)
		return fmt.Errorf("CodeSource not exists, name: %s", name)
	}

	delete(codeSourceMap, name)

	return setCodeSourceConfigMap(configMap, codeSourceMap)

}

// put
func (ov *CodeSourceHandler) PutCodeSourceToConfigMap(codeSource model.CodeSource) error {
	configMap, err := getOrCreateCodeSourceConfigMap()
	if err != nil {
		return err
	}

	codeSourceMap, err := getCodeSourceMap(configMap)
	if err != nil {
		return err
	}

	codeSource.CreateTime = codeSourceMap[codeSource.Name].CreateTime

	codeSourceMap[codeSource.Name] = codeSource

	return setCodeSourceConfigMap(configMap, codeSourceMap)
}

// get
func (ov *CodeSourceHandler) GetCodeSourceFromConfigMap(name string) (model.CodeSource, error) {
	if len(name) == 0 {
		return model.CodeSource{}, fmt.Errorf("name is empty")
	}

	configMap, err := getOrCreateCodeSourceConfigMap()
	if err != nil {
		return model.CodeSource{}, err
	}

	codeSourceMap, err := getCodeSourceMap(configMap)
	if err != nil {
		return model.CodeSource{}, err
	}

	codeSource, exists := codeSourceMap[name]
	if !exists {
		klog.Errorf("CodeSource not exists, userID: %s", name)
		return model.CodeSource{}, fmt.Errorf("CodeSource not exists, userID: %s", name)
	}

	return codeSource, nil
}

// get all
func (ov *CodeSourceHandler) ListCodeSourceFromConfigMap() (model.CodeSourceMap, error) {
	configMap, err := getOrCreateCodeSourceConfigMap()
	if err != nil {
		return model.CodeSourceMap{}, err
	}

	codeSourceMap, err := getCodeSourceMap(configMap)
	if err != nil {
		return model.CodeSourceMap{}, err
	}

	return codeSourceMap, nil
}

// set
func setCodeSourceConfigMap(configMap *v1.ConfigMap, codeSourceMap model.CodeSourceMap) error {

	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return fmt.Errorf("ConfigMap is nil")
	}

	codeSourceMapBytes, err := json.Marshal(codeSourceMap)
	if err != nil {
		klog.Errorf("CodeSourceMap Marshal failed, err: %v", err)
	}

	configMap.Data[CodesourceConfigMapKey] = string(codeSourceMapBytes)

	rs := clientmgr.GetCtrlClient().Update(context.TODO(), configMap)

	return rs
}

func getOrCreateCodeSourceConfigMap() (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := clientmgr.GetCtrlClient().Get(context.TODO(),
		apitypes.NamespacedName{
			Namespace: constants.KubeDLSystemNamespace,
			Name:      CodesourceConfigMapName,
		}, configMap)

	// Create initial user info ConfigMap if not exists
	if errors.IsNotFound(err) {
		initConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KubeDLSystemNamespace,
				Name:      CodesourceConfigMapName,
			},
			Data: map[string]string{
				CodesourceConfigMapKey: "{}",
			},
		}
		_ = clientmgr.GetCtrlClient().Create(context.TODO(), initConfigMap)
		return initConfigMap, nil
	} else if err != nil {
		klog.Errorf("Failed to get ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, CodesourceConfigMapName, err)
		return configMap, err
	}

	return configMap, nil
}

func getCodeSourceMap(configMap *v1.ConfigMap) (model.CodeSourceMap, error) {

	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return model.CodeSourceMap{}, fmt.Errorf("ConfigMap is nil")
	}

	codesources, exists := configMap.Data[CodesourceConfigMapKey]
	if !exists {
		klog.Errorf("ConfigMap key `%s` not exists", CodesourceConfigMapKey)
		return model.CodeSourceMap{}, fmt.Errorf("ConfigMap key `%s` not exists", CodesourceConfigMapKey)
	}
	if len(codesources) == 0 {
		klog.Warningf("CodeSources is empty")
		return model.CodeSourceMap{}, nil
	}

	codeSourceMap := model.CodeSourceMap{}
	err := json.Unmarshal([]byte(codesources), &codeSourceMap)
	if err != nil {
		klog.Errorf("ConfigMap json Unmarshal error, content: %s, err: %v", codesources, err)
		return codeSourceMap, err
	}

	return codeSourceMap, nil
}
