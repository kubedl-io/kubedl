package auth

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

type emptyAuth struct {
	defaultLoginID   string
	defaultLoginName string
}

func NewEmptyAuth() Auth {
	return &emptyAuth{
		defaultLoginID: "EmptyLoginID",
		defaultLoginName: "Anonymous",
	}
}

func (auth *emptyAuth) Login(c *gin.Context) error {
	session := sessions.Default(c)
	session.Set(SessionKeyLoginID, auth.defaultLoginID)
	session.Set(SessionKeyLoginName, auth.defaultLoginName)
	return session.Save()
}

func (auth *emptyAuth) Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionKeyLoginID)
	session.Delete(SessionKeyLoginName)
	return session.Save()
}

func (auth emptyAuth) Authorize(c *gin.Context) error {
	session := sessions.Default(c)
	v := session.Get(SessionKeyLoginID)
	if v == nil || v.(string) != auth.defaultLoginID {
		klog.Warningf("Authorize failed")
		return auth.Login(c)
	}
	return nil
}
