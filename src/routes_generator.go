package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/crewjam/saml/samlsp"
	"net/http"
	"net/url"
	"os"
)

type samlHosts []string

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

	rootURL, err := url.Parse(domain)
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

func (g RouteGenerator) IsAllowedHost(host string) (bool, error) {
	hostsFromEnv := os.Getenv("SAML_HOSTS")
	hosts := samlHosts{}

	if err := json.Unmarshal([]byte(hostsFromEnv), &hosts); err != nil {
		return false, err
	}

	return contains(hosts, host), nil
}

func (g RouteGenerator) MakeRedirectUrl(r *http.Request) string {
	fmt.Printf("%+v\n", r.Header)
	protocol := r.Header.Get("x-forwarded-for")
	u := url.URL{
		Scheme:      protocol,
		Opaque:      r.URL.Opaque,
		User:        r.URL.User,
		Host:        r.Host,
		Path:        r.RequestURI,
		RawPath:     r.URL.RawPath,
		ForceQuery:  r.URL.ForceQuery,
		RawQuery:    r.URL.RawQuery,
		Fragment:    r.URL.Fragment,
		RawFragment: r.URL.RawFragment,
	}

	fmt.Println(u.String())
	fmt.Println(r.URL.String())

	return u.String()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}