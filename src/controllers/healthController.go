package controllers

import (
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/gin-gonic/gin"
)

type HealthController struct {
	Router *gin.Engine
	Logger sharedKernel.Logger
}

func (c HealthController) Handler() gin.IRoutes {
	return c.Router.GET("/", func(context *gin.Context) {
		context.Status(200)
	})
}
