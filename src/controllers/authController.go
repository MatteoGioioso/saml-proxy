package controllers

import (
	"github.com/MatteoGioioso/saml-proxy/director"
	"github.com/MatteoGioioso/saml-proxy/domain"
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	Router     *gin.Engine
	SamlDomain *domain.SamlDomain
	Logger     sharedKernel.Logger
	Director   director.Director
}

func (c AuthController) Handler() gin.IRoutes {
	return c.Router.GET("/saml/auth", func(context *gin.Context) {
		rootUrl, err := c.Director.GetRootUrl(context.Request)
		if err != nil {
			c.Logger.Failure(err)
			context.JSON(400, gin.H{"message": err.Error()})
			return
		}

		samlSP, err := c.SamlDomain.GetProvider(rootUrl)
		if err != nil {
			c.Logger.Failure(err)
			context.JSON(400, gin.H{"message": err.Error()})
			return
		}

		session, err := samlSP.Session.GetSession(context.Request)
		if session != nil {
			context.Status(200)
			return
		}

		if err == samlsp.ErrNoSession {
			context.Status(401)
			return
		}
	})
}
