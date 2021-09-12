package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/crewjam/saml/samlsp"
	"net/http"
	"net/url"
)

type Server struct {
}

func (s Server) Saml() (*samlsp.Middleware, error) {
	keyPair, err := tls.LoadX509KeyPair("assets/saml-proxy.crt", "assets/saml-proxy.key")
	if err != nil {
		return nil, err
	}

	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, err
	}

	idpMetadataURL, err := url.Parse("https://dev-9as5u1kw.jp.auth0.com/samlp/metadata/wcoGeniugLE0w2hA2qBdJENDi1Rt63A4?_gl=1*e25zaz*rollup_ga*MTMwOTI3MTQ5MC4xNjMxMzUyODY5*rollup_ga_F1G3E656YZ*MTYzMTQyNTI5MC4zLjEuMTYzMTQyNTI5OC41Mg..&_ga=2.237393091.259649310.1631352871-1309271490.1631352869")
	if err != nil {
		return nil, err
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		return nil, err
	}

	rootURL, err := url.Parse("http://localhost:9000")
	if err != nil {
		return nil, err
	}

	samlSP, err := samlsp.New(samlsp.Options{
		EntityID:          "SAMLProxy",
		URL:               *rootURL,
		Key:               keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate:       keyPair.Leaf,
		AllowIDPInitiated: true,
		IDPMetadata:       idpMetadata,
		SignRequest:       true,
	})
	if err != nil {
		return nil, err
	}

	return samlSP, nil
}
