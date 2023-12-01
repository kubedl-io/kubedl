package auth

import (
	"errors"
	"flag"

	"github.com/gin-gonic/gin"
)

const (
	SessionKeyLoginID = "loginId"
)

func init() {
	authTypes := map[string]AuthRegister{
		"none":   NewEmptyAuth,
		"config": NewConfigAuth,
	}

	var authType string
	flag.StringVar(&authType, "authentication-mode", "none",
		"set authentication mode. By default, no authentication . Use --authentication-mode=none to explicitly disable authentication,"+
			" --authentication-mode=config to authenticate using configMap")
	flag.Parse()
	GetAuth = authTypes[authType]
}

var (
	GetAuth AuthRegister

	ErrLoginInvalid = errors.New("login id is invalid")
	ErrGetAuthError = errors.New("get oauthInfo error")
)

type AuthRegister func() Auth

type Auth interface {
	Login(c *gin.Context) error
	Logout(c *gin.Context) error
	Authorize(c *gin.Context) error
}
