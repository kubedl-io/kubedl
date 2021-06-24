package auth

import (
	"errors"
	"flag"
	"github.com/gin-gonic/gin"
)

const (
	SessionKeyLoginID   = "loginId"
	SessionKeyLoginName = "loginName"
)

func init() {
	authTypes := map[string]AuthRegister {
		"none": NewEmptyAuth,
		"config": NewConfigAuth,
	}

	var authType string
	flag.StringVar(&authType, "auth-type", "none",
		"set authorize middleware name, --auth-type=none to disable authorize," +
		" --auth-type=config to authorize from configMap")
	flag.Parse()
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
}


