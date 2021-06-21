package middleware

import (
	"net/http"
	"strings"

	"github.com/alibaba/kubedl/console/backend/pkg/auth"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
	"github.com/gin-gonic/gin"

	"k8s.io/klog"
)

// CheckAuthMiddleware check auth
func CheckAuthMiddleware(loginAuth auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error

		defer func() {
			if err != nil {
				klog.Errorf("[check auth] check auth failed, err: %v", err)
			}
		}()

		// Skip static js and css files checking auth.
		if c.Request.URL != nil && (strings.HasSuffix(c.Request.URL.Path, ".js") ||
			strings.HasSuffix(c.Request.URL.Path, ".css")) {
			c.Next()
			return
		}

		if c.Request.URL != nil && (strings.HasPrefix(c.Request.URL.Path, constants.ApiV1Routes+"/login") ||
			strings.HasPrefix(c.Request.URL.Path, constants.ApiV1Routes+"/ingressAuth") ||
			strings.HasPrefix(c.Request.URL.Path, "/403") ||
			strings.HasPrefix(c.Request.URL.Path, "/404") ||
			strings.HasPrefix(c.Request.URL.Path, "/500")) {
			klog.Infof("[check auth] request prefixed with %s, go next.", c.Request.URL.Path)
			c.Next()
			return
		}

		err = loginAuth.Authorize(c)
		if err == nil {
			c.Next()
			return
		}else if err == auth.GetAuthError {
			klog.Errorf("[check auth] getOauthInfo err, url: %s, err: %v", c.FullPath(), err)
			utils.Redirect500(c)
			c.Abort()
			return
		}

		loginUrl, err := loginAuth.GetLoginUrl(c)
		if err != nil {
			klog.Errorf("[check auth] loginUrl get url error, err: %v", err)
			utils.Redirect403(c)
			c.Abort()
			return
		}
		c.Redirect(http.StatusFound, loginUrl)
		c.Abort()
	}
}
