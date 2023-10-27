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
	DatasourceConfigMapName = "kubedl-datasource-config"
	DatasourceConfigMapKey  = "datasource"
)

func NewDataSourceHandler() *DataSourceHandler {
	return &DataSourceHandler{client: clientmgr.GetCtrlClient()}
}

type DataSourceHandler struct {
	client client.Client
}

// post
func (ov *DataSourceHandler) PostDataSourceToConfigMap(dataSource model.DataSource) error {
	klog.Infof("DataSource : %s", dataSource)

	configMap, err := getOrCreateDataSourceConfigMap()
	if err != nil {
		return err
	}

	dataSourceMap, err := getDataSourceMap(configMap)
	if err != nil {
		return err
	}

	_, exists := dataSourceMap[dataSource.Name]
	if exists {
		klog.Errorf("DataSource exists, name: %s", dataSource.Name)
		return fmt.Errorf("DataSource exists, name: %s", dataSource.Name)
	}

	dataSourceMap[dataSource.Name] = dataSource

	return setDataSourceConfigMap(configMap, dataSourceMap)
}

// delete
func (ov *DataSourceHandler) DeleteDataSourceFromConfigMap(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is empty")
	}

	configMap, err := getOrCreateDataSourceConfigMap()
	if err != nil {
		return err
	}

	dataSourceMap, err := getDataSourceMap(configMap)
	if err != nil {
		return err
	}

	_, exists := dataSourceMap[name]
	if !exists {
		klog.Errorf("DataSource not exists, name: %s", name)
		return fmt.Errorf("DataSource not exists, name: %s", name)
	}

	delete(dataSourceMap, name)

	return setDataSourceConfigMap(configMap, dataSourceMap)

}

// put
func (ov *DataSourceHandler) PutDataSourceToConfigMap(dataSource model.DataSource) error {

	configMap, err := getOrCreateDataSourceConfigMap()
	if err != nil {
		return err
	}

	dataSourceMap, err := getDataSourceMap(configMap)
	if err != nil {
		return err
	}

	dataSource.CreateTime = dataSourceMap[dataSource.Name].CreateTime

	dataSourceMap[dataSource.Name] = dataSource

	rs := setDataSourceConfigMap(configMap, dataSourceMap)

	return rs
}

// get
func (ov *DataSourceHandler) GetDataSourceFromConfigMap(name string) (model.DataSource, error) {
	if len(name) == 0 {
		return model.DataSource{}, fmt.Errorf("name is empty")
	}

	configMap, err := getOrCreateDataSourceConfigMap()
	if err != nil {
		return model.DataSource{}, err
	}

	dataSourceMap, err := getDataSourceMap(configMap)
	if err != nil {
		return model.DataSource{}, err
	}

	dataSource, exists := dataSourceMap[name]
	if !exists {
		klog.Errorf("DataSource not exists, userID: %s", name)
		return model.DataSource{}, fmt.Errorf("DataSource not exists, userID: %s", name)
	}

	return dataSource, nil
}

// get all
func (ov *DataSourceHandler) ListDataSourceFromConfigMap() (model.DataSourceMap, error) {
	configMap, err := getOrCreateDataSourceConfigMap()
	if err != nil {
		return model.DataSourceMap{}, err
	}

	dataSourceMap, err := getDataSourceMap(configMap)
	if err != nil {
		return model.DataSourceMap{}, err
	}

	return dataSourceMap, nil
}

// set
func setDataSourceConfigMap(configMap *v1.ConfigMap, dataSourceMap model.DataSourceMap) error {
	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return fmt.Errorf("ConfigMap is nil")
	}

	dataSourceMapBytes, err := json.Marshal(dataSourceMap)
	if err != nil {
		klog.Errorf("DataSourceMap Marshal failed, err: %v", err)
	}

	configMap.Data[DatasourceConfigMapKey] = string(dataSourceMapBytes)

	return clientmgr.GetCtrlClient().Update(context.TODO(), configMap)
}

func getOrCreateDataSourceConfigMap() (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := clientmgr.GetCtrlClient().Get(context.TODO(),
		apitypes.NamespacedName{
			Namespace: constants.KubeDLSystemNamespace,
			Name:      DatasourceConfigMapName,
		}, configMap)

	// Create initial user info ConfigMap if not exists
	if errors.IsNotFound(err) {
		initConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.KubeDLSystemNamespace,
				Name:      DatasourceConfigMapName,
			},
			Data: map[string]string{
				DatasourceConfigMapKey: "{}",
			},
		}
		if err := clientmgr.GetCtrlClient().Create(context.TODO(), initConfigMap); err != nil {
			klog.Errorf("Failed to create ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, DatasourceConfigMapName, err)
			return nil, err
		}
		return initConfigMap, nil
	} else if err != nil {
		klog.Errorf("Failed to get ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, DatasourceConfigMapName, err)
		return configMap, err
	}
	return configMap, nil
}

func getDataSourceMap(configMap *v1.ConfigMap) (model.DataSourceMap, error) {

	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return model.DataSourceMap{}, fmt.Errorf("ConfigMap is nil")
	}

	datasources, exists := configMap.Data[DatasourceConfigMapKey]
	if !exists {
		klog.Errorf("ConfigMap key `%s` not exists", DatasourceConfigMapKey)
		return model.DataSourceMap{}, fmt.Errorf("ConfigMap key `%s` not exists", DatasourceConfigMapKey)
	}
	if len(datasources) == 0 {
		klog.Warningf("DataSources is empty")
		return model.DataSourceMap{}, nil
	}

	dataSourceMap := model.DataSourceMap{}
	err := json.Unmarshal([]byte(datasources), &dataSourceMap)
	if err != nil {
		klog.Errorf("ConfigMap json Unmarshal error, content: %s, err: %v", datasources, err)
		return dataSourceMap, err
	}

	return dataSourceMap, nil
}
