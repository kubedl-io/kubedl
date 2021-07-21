package utils

import (
	"github.com/gin-gonic/gin"
)

func Redirect403(c *gin.Context) {
	c.JSON(403, nil)
}

func Redirect404(c *gin.Context) {
	c.JSON(404, nil)
}

func Redirect500(c *gin.Context) {
	c.JSON(500, nil)
}
