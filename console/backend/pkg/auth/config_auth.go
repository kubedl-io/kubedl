package auth

import (
	"bytes"
	"encoding/json"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"k8s.io/klog"
)

const configMapKeyUsers = "users"

type configAuth struct {}

func NewConfigAuth() Auth {
	return &configAuth{}
}

func (auth *configAuth) Login(c *gin.Context) error {
	var loginData struct{
		Username string
		Password string
	}
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	c.Request.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
	data := buf[0:n]
	if err := json.Unmarshal(data, &loginData);err != nil {
		klog.Error("request form error")
		return LoginInvalid
	}
	config, err := model.GetOrCreateUserInfoConfigMap()
	if err != nil {
		klog.Errorf("get user configMap error: %v", err)
		return GetAuthError
	}
	users, exists := config.Data[configMapKeyUsers]
	if !exists || len(users) == 0 {
		klog.Errorf("ConfigMap key `%s` not exists", configMapKeyUsers)
		return GetAuthError
	}
	var userInfos map[string]map[string]string
	if err := json.Unmarshal([]byte(users), &userInfos); err != nil {
		return GetAuthError
	}
	if userInfo, exist := userInfos[loginData.Username]; exist {
		if userInfo["password"] != loginData.Password {
			return LoginInvalid
		}
		session := sessions.Default(c)
		session.Set(SessionKeyLoginID, userInfo["uid"])
		session.Set(SessionKeyLoginName, userInfo["login_name"])
		return session.Save()

	}
	return LoginInvalid
}

func (auth *configAuth) Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionKeyLoginID)
	session.Delete(SessionKeyLoginName)
	return session.Save()
}

func (auth configAuth) Authorize(c *gin.Context) error {
	session := sessions.Default(c)
	v := session.Get(SessionKeyLoginID)
	if v == nil {
		klog.Warningf("authorize logout")
		return LoginInvalid
	}
	info, err := model.GetUserInfoFromConfigMap(v.(string))
	if err != nil || session.Get(SessionKeyLoginName) != info.LoginName {
		klog.Warningf("Authorize failed")
		return LoginInvalid
	}
	return nil
}

