package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Redirect403(c *gin.Context) {
	c.Redirect(http.StatusFound, "/403")
}

func Redirect404(c *gin.Context) {
	c.Redirect(http.StatusFound, "/404")
}

func Redirect500(c *gin.Context) {
	c.Redirect(http.StatusFound, "/500")
}
