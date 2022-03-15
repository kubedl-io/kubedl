package model

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/kubedl/console/backend/pkg/constants"

	v1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"
)

type UserInfo struct {
	// Uid login account id
	Username string `json:"username"`
	Password string `json:"password"`
}

const (
	configMapKeyUsers = "users"
)

func GetUserInfoFromConfigMap(username string) (UserInfo, error) {
	if len(username) == 0 {
		return UserInfo{}, fmt.Errorf("username is empty")
	}

	configMap, err := GetUserInfoConfigMap()
	if err != nil {
		return UserInfo{}, err
	}

	userInfos, err := getUserInfos(configMap)
	if err != nil {
		return UserInfo{}, err
	}
	for _, userInfo := range userInfos {
		if userInfo.Username == username {
			return userInfo, nil
		}
	}
	klog.Errorf("UserInfo not exists, username: %s", username)
	return UserInfo{}, fmt.Errorf("UserInfo not exists, username: %s", username)
}

func GetUserInfoConfigMap() (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	err := clientmgr.GetCtrlClient().Get(context.TODO(),
		apitypes.NamespacedName{
			Namespace: constants.KubeDLSystemNamespace,
			Name:      constants.KubeDLConsoleConfig,
		}, configMap)

	if err != nil {
		klog.Errorf("Failed to get ConfigMap, ns: %s, name: %s, err: %v", constants.KubeDLSystemNamespace, constants.KubeDLConsoleConfig, err)
		return nil, err
	}

	return configMap, nil
}

func getUserInfos(configMap *v1.ConfigMap) ([]UserInfo, error) {
	if configMap == nil {
		klog.Errorf("ConfigMap is nil")
		return nil, fmt.Errorf("ConfigMap is nil")
	}

	users, exists := configMap.Data[configMapKeyUsers]
	if !exists {
		klog.Errorf("ConfigMap key `%s` not exists", configMapKeyUsers)
		return nil, fmt.Errorf("ConfigMap key `%s` not exists", configMapKeyUsers)
	}

	var userInfos []UserInfo
	err := json.Unmarshal([]byte(users), &userInfos)
	if err != nil {
		klog.Errorf("ConfigMap json Unmarshal error, content: %s, err: %v", users, err)
		return nil, err
	}

	return userInfos, nil
}
