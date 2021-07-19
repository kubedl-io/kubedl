package model

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"

	clientmgr "github.com/alibaba/kubedl/pkg/storage/backends/client"
	v1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

type UserInfo struct {
	// Uid login account id
	Uid string `json:"uid"`
	// LoginName login account name
	LoginName string `json:"login_name"`
	//
	Password string `json:"password"`
}

type UserInfoMap map[string]UserInfo

const (
	configMapKeyUsers = "users"
)

func GetUserInfoFromConfigMap(userID string) (UserInfo, error) {
	if len(userID) == 0 {
		return UserInfo{}, fmt.Errorf("userID is empty")
	}

	configMap, err := GetUserInfoConfigMap()
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

func GetUserInfoConfigMap() (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := clientmgr.GetCtrlClient().Get(context.TODO(),
		apitypes.NamespacedName{
			Namespace: constants.KubeDLSystemNamespace,
			Name:      constants.ConfigMapName,
		}, configMap)

	if err != nil {
		klog.Errorf("Failed to get ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, constants.ConfigMapName, err)
		return nil, err
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
