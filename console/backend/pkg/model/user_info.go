package model

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"

	clientmgr "github.com/alibaba/kubedl/pkg/storage/backends/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

type UserInfo struct {

	// aliCloud main account id
	Aid string `json:"aid"`
	// aliCloud login account id
	Uid string `json:"uid"`
	// the name displayed by the logged-in user
	Name string `json:"name"`
	// aliCloud login account
	LoginName string `json:"login_name"`
	// aliCloud ram  login account
	Upn string `json:"upn"`
}

type UserInfoMap map[string]UserInfo

const (
	configMapName      = "user-info-config"
	configMapKeyUsers  = "users"
)

func StoreUserInfoToConfigMap(userInfo UserInfo) error {
	configMap, err := getOrCreateUserInfoConfigMap()
	if err != nil {
		return err
	}

	userInfoMap, err := getUserInfoMap(configMap)
	if err != nil {
		return err
	}

	userInfoMap[userInfo.Uid] = userInfo

	return updateUserInfoConfigMap(configMap, userInfoMap)
}

func GetUserInfoFromConfigMap(userID string) (UserInfo, error) {
	if len(userID) == 0 {
		return UserInfo{}, fmt.Errorf("userID is empty")
	}

	configMap, err := getOrCreateUserInfoConfigMap()
	if err != nil {
		return UserInfo{}, err
	}

	userInfoMap, err := getUserInfoMap(configMap)
	if err != nil {
		return UserInfo{}, err
	}

	userInfo, exists := userInfoMap[userID]
	if !exists {
		klog.Errorf("UserInfo not exists, userID: %s", userID)
		return UserInfo{}, fmt.Errorf("UserInfo not exists, userID: %s", userID)
	}

	return userInfo, nil
}

func getOrCreateUserInfoConfigMap() (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := clientmgr.GetCtrlClient().Get(context.TODO(),
		apitypes.NamespacedName{
			Namespace: constants.DLCSystemNamespace,
			Name:      configMapName,
		}, configMap)

	// Create initial user info ConfigMap if not exists
	if errors.IsNotFound(err) {
		initConfigMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.DLCSystemNamespace,
				Name:      configMapName,
			},
			Data: map[string]string{
				configMapKeyUsers: "{}",
			},
		}
		clientmgr.GetCtrlClient().Create(context.TODO(), initConfigMap)
		return initConfigMap, nil
	} else if err != nil {
		klog.Errorf("Failed to get ConfigMap, ns: %s, name: %s, err: %v", constants.DLCSystemNamespace, configMapName, err)
		return configMap, err
	}

	return configMap, nil
}

func updateUserInfoConfigMap(configMap *v1.ConfigMap, userInfoMap UserInfoMap) error {
	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return fmt.Errorf("ConfigMap is nil")
	}

	userInfoMapBytes, err := json.Marshal(userInfoMap)
	if err != nil {
		klog.Errorf("UserInfoMap Marshal failed, err: %v", err)
	}

	configMap.Data[configMapKeyUsers] = string(userInfoMapBytes)

	return clientmgr.GetCtrlClient().Update(context.TODO(), configMap)
}

func getUserInfoMap(configMap *v1.ConfigMap) (UserInfoMap, error) {
	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return UserInfoMap{}, fmt.Errorf("ConfigMap is nil")
	}

	users, exists := configMap.Data[configMapKeyUsers]
	if !exists {
		klog.Errorf("ConfigMap key `%s` not exists", configMapKeyUsers)
		return UserInfoMap{}, fmt.Errorf("ConfigMap key `%s` not exists", configMapKeyUsers)
	}
	if len(users) == 0 {
		klog.Warningf("UserInfoMap is empty")
		return UserInfoMap{}, nil
	}

	userInfoMap := UserInfoMap{}
	err := json.Unmarshal([]byte(users), &userInfoMap)
	if err != nil {
		klog.Errorf("ConfigMap json Unmarshal error, content: %s, err: %v", users, err)
		return userInfoMap, err
	}

	return userInfoMap, nil
}
