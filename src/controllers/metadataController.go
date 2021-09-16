package controllers

import (
	"github.com/MatteoGioioso/saml-proxy/director"
	"github.com/MatteoGioioso/saml-proxy/domain"
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/gin-gonic/gin"
)

type MetadataController struct {
	Router     *gin.Engine
	SamlDomain *domain.SamlDomain
	Logger     sharedKernel.Logger
	Director   director.Director
}

func (c MetadataController) Handler() gin.IRoutes {
	return c.Router.GET("/saml/metadata", func(context *gin.Context) {
		rootUrl, err := c.Director.GetRootUrl(context.Request)
		if err != nil {
			context.JSON(400, gin.H{"message": err})
			return
		}

		middleware, err := c.SamlDomain.GetProvider(rootUrl)
		if err != nil {
			context.JSON(400, gin.H{"message": err})
			return
		}

		middleware.ServeHTTP(context.Writer, context.Request)
	})
}
