package auth

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
	"net/url"
)

type emptyAuth struct {
	oauthPath        string
	defaultAccountID string
	defaultLoginID   string
}

func NewEmptyAuth() Auth {
	return &emptyAuth{
		oauthPath: "/login/oauth2/empty",
		defaultAccountID: "EmptyAccountID",
		defaultLoginID: "EmptyLoginID",
	}
}

func (auth *emptyAuth) Login(c *gin.Context) error {
	loginType := c.Param("type")
	if loginType != "empty" {
		klog.Errorf("login not support current login type, url: %s", c.FullPath())
		return errors.New("invalid authorize type")
	}

	session := sessions.Default(c)
	session.Set(SessionKeyAccountID, auth.defaultAccountID)
	session.Set(SessionKeyLoginID, auth.defaultLoginID)
	return session.Save()
}

func (auth *emptyAuth) Logout(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionKeyAccountID)
	session.Delete(SessionKeyLoginID)
	return session.Save()
}

func (auth emptyAuth) Authorize(c *gin.Context) error {
	session := sessions.Default(c)
	v := session.Get(SessionKeyAccountID)
	if v == nil || v.(string) != auth.defaultAccountID {
		klog.Warningf("Authorize failed")
		return LoginInvalid
	}
	return nil
}

func (auth emptyAuth) GetLoginUrl(c *gin.Context) (loginUrl string, err error) {
	vals := url.Values{}
	vals.Add("redirect_uri", c.Request.URL.String())
	rurl, _ := url.Parse("http://" + c.Request.Host + auth.oauthPath)
	rurl.RawQuery = vals.Encode()
	return rurl.String(), nil
}