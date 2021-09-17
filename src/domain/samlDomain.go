package domain

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/crewjam/saml/samlsp"
	"log"
	"net/http"
	"net/url"
	"os"
)

type samlHosts []string

type SamlDomain struct {
	SamlMiddlewares  map[string]*samlsp.Middleware
	MetadataEndpoint string
	AllowedHosts     []string
	logger           sharedKernel.Logger
}

func NewSamlDomain(metadataEndpoint string, logger sharedKernel.Logger) *SamlDomain {
	s := &SamlDomain{}
	hosts, err := s.getAllowedHosts()
	if err != nil {
		log.Fatal(err)
	}

	s.AllowedHosts = hosts
	s.SamlMiddlewares = make(map[string]*samlsp.Middleware)
	s.MetadataEndpoint = metadataEndpoint
	s.logger = logger

	return s
}

func (g SamlDomain) CreateMiddlewares() error {
	for _, host := range g.AllowedHosts {
		g.logger.Info("Creating middlewares for host: " + host)
		middleware, err := g.createMiddleware(host)
		if err != nil {
			return err
		}

		g.SamlMiddlewares[host] = middleware
	}

	return nil
}

func (g SamlDomain) createMiddleware(domain string) (*samlsp.Middleware, error) {
	if os.Getenv("SSL_CERTIFICATE_AUTOGENERATE") == "true" {
		g.logger.Info("Generating certificates for host: " + domain)
		if err := sharedKernel.GenerateCertificates(domain); err != nil {
			return nil, err
		}
	}

	certPath := sharedKernel.GetCertPath(domain)
	certKeyPath := sharedKernel.GetCertKeyPath(domain)
	if certPath == "" || certKeyPath == "" {
		return nil, errors.New("cert or key path is missing")
	}
	keyPair, err := tls.LoadX509KeyPair(certPath, certKeyPath)
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

	rootURL, err := url.Parse(fmt.Sprintf("https://%s", domain))
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

func (g SamlDomain) GetProvider(domain string) (*samlsp.Middleware, error) {
	parse, err := url.Parse(domain)
	if err != nil {
		return nil, err
	}

	host, ok := g.SamlMiddlewares[parse.Host]
	if ok {
		return host, nil
	}

	return nil, errors.New("no allowed host found")
}

func (g SamlDomain) getAllowedHosts() ([]string, error) {
	hostsFromEnv := os.Getenv("SAML_HOSTS")
	hosts := samlHosts{}

	if err := json.Unmarshal([]byte(hostsFromEnv), &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}
