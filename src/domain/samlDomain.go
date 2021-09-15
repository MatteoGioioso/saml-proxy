package domain

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/crewjam/saml/samlsp"
	"log"
	"net/http"
	"net/url"
	"os"
)

type samlHosts []string

type SamlDomain struct {
	SamlMiddlewares map[string]*samlsp.Middleware
	MetadataEndpoint string
	AllowedHosts []string
}

func NewSamlDomain(metadataEndpoint string) *SamlDomain {
	s := &SamlDomain{}
	hosts, err := s.getAllowedHosts()
	if err != nil {
		log.Fatal(err)
	}

	s.AllowedHosts = hosts
	s.SamlMiddlewares = make(map[string]*samlsp.Middleware)
	s.MetadataEndpoint = metadataEndpoint

	return s
}

func (g SamlDomain) CreateMiddlewares() error {
	for _, host := range g.AllowedHosts {
		middleware, err := g.createMiddleware(host)
		if err != nil {
			return err
		}
		
		g.SamlMiddlewares[host] = middleware
	}

	return nil
}

func (g SamlDomain) createMiddleware(domain string) (*samlsp.Middleware, error) {
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

func (g SamlDomain) GetProvider(domain string) *samlsp.Middleware {
	host, ok := g.SamlMiddlewares[domain]
	if ok {
		return host
	}

	// TODO: change with proper 500 response
	log.Fatal("no allowed host")
	return nil
}

func (g SamlDomain) getAllowedHosts() ([]string, error) {
	hostsFromEnv := os.Getenv("SAML_HOSTS")
	hosts := samlHosts{}

	if err := json.Unmarshal([]byte(hostsFromEnv), &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}