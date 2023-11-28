package api

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"

	"github.com/alibaba/kubedl/console/backend/pkg/auth"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
)

func NewAuthAPIsController(loginAuth auth.Auth) *authAPIsController {
	return &authAPIsController{
		loginAuth: loginAuth,
	}
}

type authAPIsController struct {
	loginAuth auth.Auth
}

func (ac *authAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	routes.GET("/login/oauth2", ac.login)
	routes.POST("/login/oauth2", ac.login)
	routes.GET("/current-user", ac.currentUser)
	routes.GET("/logout", ac.logout)
	routes.GET("/ingressAuth", ac.ingressAuth)
}

// login authorize and redirect to home page
func (ac *authAPIsController) login(c *gin.Context) {
	err := ac.loginAuth.Login(c)
	if err != nil {
		klog.Errorf("login err, err: %v, url: %s", err, c.FullPath())
		if err == auth.ErrLoginInvalid {
			utils.Redirect403(c)
		} else {
			utils.Redirect500(c)
		}
		return
	}
	utils.Succeed(c, nil)
}

func (ac *authAPIsController) logout(c *gin.Context) {
	err := ac.loginAuth.Logout(c)
	if err != nil {
		klog.Errorf("logout err, err: %v, url: %s", err, c.FullPath())
		utils.Redirect500(c)
	}
	utils.Succeed(c, nil)
}

func (ac *authAPIsController) currentUser(c *gin.Context) {
	session := sessions.Default(c)
	utils.Succeed(c, map[string]interface{}{
		"loginId": session.Get(auth.SessionKeyLoginID),
	})
}

func (ac *authAPIsController) ingressAuth(c *gin.Context) {
	err := ac.loginAuth.Authorize(c)
	if err != nil {
		if err == auth.ErrLoginInvalid {
			klog.Errorf("user account error, err: %v", auth.ErrLoginInvalid)
			utils.Redirect403(c)
		} else {
			klog.Errorf("authorize error, err: %v", err)
			utils.Redirect500(c)
		}
	}
	utils.Succeed(c, nil)
}
