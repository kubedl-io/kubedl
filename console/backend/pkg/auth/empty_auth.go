package auth

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

type emptyAuth struct {
	defaultLoginID string
}

func NewEmptyAuth() Auth {
	return &emptyAuth{
		defaultLoginID: "Anonymous",
	}
}

func (auth *emptyAuth) Login(c *gin.Context) error {
	session := sessions.Default(c)
	session.Set(SessionKeyLoginID, auth.defaultLoginID)
	return session.Save()
}

func (auth *emptyAuth) Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionKeyLoginID)
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
