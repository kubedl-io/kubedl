package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alibaba/kubedl/console/backend/pkg/handlers"
	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/console/backend/pkg/utils"
)

func NewCodeSourceAPIsController() *codeSourceAPIsController {
	return &codeSourceAPIsController{
		codeSourceHandler: handlers.NewCodeSourceHandler(),
	}
}

type codeSourceAPIsController struct {
	codeSourceHandler *handlers.CodeSourceHandler
}

func (dc *codeSourceAPIsController) RegisterRoutes(routes *gin.RouterGroup) {
	overview := routes.Group("/codesource")
	overview.POST("", dc.postCodeSource)
	overview.DELETE("/:name", dc.deleteCodeSource)
	overview.PUT("", dc.putCodeSource)
	overview.GET("", dc.getCodeSource)
	overview.GET("/:name", dc.getCodeSource)
}

func (dc *codeSourceAPIsController) postCodeSource(c *gin.Context) {
	userId := c.PostForm("userid")
	username := c.PostForm("username")
	name := c.PostForm("name")
	_type := c.PostForm("type")
	codePath := c.PostForm("code_path")
	defaultBranch := c.PostForm("default_branch")
	localPath := c.PostForm("local_path")
	description := c.PostForm("description")
	createTime := time.Now().Format("2006-01-02 15:04:05")
	updateTime := time.Now().Format("2006-01-02 15:04:05")

	codesource := model.CodeSource{
		UserId:        userId,
		Username:      username,
		Name:          name,
		Type:          _type,
		CodePath:      codePath,
		DefaultBranch: defaultBranch,
		LocalPath:     localPath,
		Description:   description,
		CreateTime:    createTime,
		UpdateTime:    updateTime,
	}

	err := dc.codeSourceHandler.PostCodeSourceToConfigMap(codesource)
	if err != nil {
		Error(c, fmt.Sprintf("failed to createCodeSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to createCodeSource")
}

func (dc *codeSourceAPIsController) deleteCodeSource(c *gin.Context) {
	name := c.Param("name")
	err := dc.codeSourceHandler.DeleteCodeSourceFromConfigMap(name)
	if err != nil {
		Error(c, fmt.Sprintf("failed to delete CodeSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to delete CodeSource.")
}

func (dc *codeSourceAPIsController) putCodeSource(c *gin.Context) {
	userId := c.PostForm("userid")
	username := c.PostForm("username")
	name := c.PostForm("name")
	_type := c.PostForm("type")
	codePath := c.PostForm("code_path")
	defaultBranch := c.PostForm("default_branch")
	localPath := c.PostForm("local_path")
	description := c.PostForm("description")
	updateTime := time.Now().Format("2006-01-02 15:04:05")

	codesource := model.CodeSource{
		UserId:        userId,
		Username:      username,
		Name:          name,
		Type:          _type,
		CodePath:      codePath,
		DefaultBranch: defaultBranch,
		LocalPath:     localPath,
		Description:   description,
		UpdateTime:    updateTime,
	}

	err := dc.codeSourceHandler.PutCodeSourceToConfigMap(codesource)
	if err != nil {
		Error(c, fmt.Sprintf("failed to putCodeSource, err=%v", err))
		return
	}
	utils.Succeed(c, "success to putCodeSource")
}

func (dc *codeSourceAPIsController) getCodeSource(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		codeSources, err := dc.codeSourceHandler.ListCodeSourceFromConfigMap()
		if err != nil {
			Error(c, fmt.Sprintf("failed to getCodeSource, err=%v", err))
			return
		}
		utils.Succeed(c, codeSources)
	} else {
		codeSources, err := dc.codeSourceHandler.GetCodeSourceFromConfigMap(name)
		if err != nil {
			Error(c, fmt.Sprintf("failed to getCodeSource, err=%v", err))
			return
		}
		utils.Succeed(c, codeSources)
	}

}
