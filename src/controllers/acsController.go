package controllers

import (
	"github.com/MatteoGioioso/saml-proxy/director"
	"github.com/MatteoGioioso/saml-proxy/domain"
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/gin-gonic/gin"
)

type AcsController struct {
	Router     *gin.Engine
	SamlDomain *domain.SamlDomain
	Logger     sharedKernel.Logger
	Director   director.Director
}

func (c AcsController) Handler() gin.IRoutes {
	return c.Router.POST("/saml/acs", func(context *gin.Context) {
		rootUrl, err := c.Director.GetRootUrl(context.Request)
		if err != nil {
			context.Writer.WriteHeader(500)
			context.Writer.Write([]byte(err.Error()))
			return
		}
		middleware := c.SamlDomain.GetProvider(rootUrl)
		middleware.ServeHTTP(context.Writer, context.Request)
	})
}
