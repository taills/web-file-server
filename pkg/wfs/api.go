package wfs

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

// Wfs Web文件服务
type Wfs struct {
	Router     *gin.Engine
	listenPort uint
}

// NewWfs 创建Web文件服务
func NewWfs(listenPort uint) *Wfs {
	r := gin.Default()
	r.POST("/*dir", func(c *gin.Context) {
		// 上传文件
		dir := c.Param("dir")
		file, err := c.FormFile("file")
	})
	return &Wfs{
		listenPort: listenPort,
		Router:     r,
	}
}

// Run 启动服务
func (w *Wfs) Run() error {
	return w.Router.Run(fmt.Sprintf(":%d", w.listenPort))
}
