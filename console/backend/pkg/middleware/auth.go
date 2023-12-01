package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/auth"
	"github.com/alibaba/kubedl/console/backend/pkg/constants"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"

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
			strings.HasPrefix(c.Request.URL.Path, constants.ApiV1Routes+"/logout") ||
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
		} else if err == auth.ErrGetAuthError {
			klog.Errorf("[check auth] getOauthInfo err, url: %s, err: %v", c.FullPath(), err)
			utils.Redirect500(c)
			c.Abort()
		} else {
			klog.Errorf("[check auth] authorize failed, err: %v", err)
			utils.Redirect403(c)
			c.Abort()
		}
		c.Abort()
	}
}
