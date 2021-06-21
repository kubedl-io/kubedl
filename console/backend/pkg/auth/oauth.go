package auth

import (
	"errors"
	"flag"
	"github.com/gin-gonic/gin"
)

const (
	SessionKeyAccountID = "accountId"
	SessionKeyLoginID   = "loginId"
	SessionKeyName      = "name"
	SessionKeyLoginName = "loginName"
)

func init() {
	authTypes := map[string]AuthRegister {
		"none": NewEmptyAuth,
	}

	var authType string
	flag.StringVar(&authType, "auth-type", "none",
		"set authorize middleware name, --authorize=none to disable authorize, default disable")

	GetAuth = authTypes[authType]
}

var (
	GetAuth AuthRegister

	LoginInvalid = errors.New("login id is inconsistent")
	GetAuthError = errors.New("get oauthInfo error")
)

type AuthRegister func() Auth

type Auth interface {
	Login(c *gin.Context) error
	Logout(c *gin.Context) error
	Authorize(c *gin.Context) error
	GetLoginUrl(c *gin.Context) (loginUrl string, err error)
}


