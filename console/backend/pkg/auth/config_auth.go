package auth

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/alibaba/kubedl/console/backend/pkg/model"
)

type configAuth struct{}

func NewConfigAuth() Auth {
	return &configAuth{}
}

func (auth *configAuth) Login(c *gin.Context) error {
	var loginData model.UserInfo
	buf := make([]byte, 1024)
	n, _ := c.Request.Body.Read(buf)
	c.Request.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
	data := buf[0:n]
	if err := json.Unmarshal(data, &loginData); err != nil {
		glog.Errorf("request form error: %v", err)
		return ErrLoginInvalid
	}
	userInfo, err := model.GetUserInfoFromConfigMap(loginData.Username)
	if err != nil {
		glog.Errorf("Get user info error: %v", err)
		return ErrLoginInvalid
	}
	if userInfo.Password != loginData.Password {
		return ErrLoginInvalid
	}
	session := sessions.Default(c)
	session.Set(SessionKeyLoginID, userInfo.Username)
	return session.Save()
}

func (auth *configAuth) Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionKeyLoginID)
	return session.Save()
}

func (auth configAuth) Authorize(c *gin.Context) error {
	session := sessions.Default(c)
	v := session.Get(SessionKeyLoginID)
	if v == nil {
		glog.Warningf("authorize logout")
		return ErrLoginInvalid
	}
	_, err := model.GetUserInfoFromConfigMap(v.(string))
	if err != nil {
		glog.Warningf("Authorize failed")
		return ErrLoginInvalid
	}
	return nil
}
