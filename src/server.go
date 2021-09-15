package main

import (
	"github.com/MatteoGioioso/saml-proxy/controllers"
	"github.com/MatteoGioioso/saml-proxy/director"
	"github.com/MatteoGioioso/saml-proxy/domain"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

var (
	metadataEndpoint = os.Getenv("SAML_METADATA_ENDPOINT")
	dir = director.Director{}
	samlDomain = domain.NewSamlDomain(metadataEndpoint)
)

func main() {
	r := gin.Default()
	if err := samlDomain.CreateMiddlewares(); err != nil {
		log.Fatal(err)
	}

	authController := controllers.AuthController{
		Router:     r,
		SamlDomain: samlDomain,
		Logger:     nil,
		Director:   dir,
	}
	signinController := controllers.SigninController{
		Router:     r,
		SamlDomain: samlDomain,
		Logger:     nil,
		Director:   dir,
	}
	acsController := controllers.AcsController{
		Router:     r,
		SamlDomain: samlDomain,
		Logger:     nil,
		Director:   dir,
	}
	metadataController := controllers.MetadataController{
		Router:     r,
		SamlDomain: samlDomain,
		Logger:     nil,
		Director:   dir,
	}

	authController.Handler()
	signinController.Handler()
	acsController.Handler()
	metadataController.Handler()

	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
