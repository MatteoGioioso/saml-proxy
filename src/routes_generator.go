package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/crewjam/saml/samlsp"
	"net/http"
	"net/url"
)

type RouteGenerator struct {
	MetadataEndpoint string
}

func (g RouteGenerator) CreateSamlService(domain string) (*samlsp.Middleware, error) {
	keyPair, err := tls.LoadX509KeyPair("assets/saml-proxy.crt", "assets/saml-proxy.key")
	if err != nil {
		return nil, err
	}

	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, err
	}

	idpMetadataURL, err := url.Parse(g.MetadataEndpoint)
	if err != nil {
		return nil, err
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		return nil, err
	}
	fmt.Println(domain)
	rootURL, err := url.Parse(fmt.Sprintf("http://%s", domain))
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
