package director

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

const (
	XForwardedProto = "X-Forwarded-Proto"
	XForwardedHost  = "X-Forwarded-Host"
	XForwardedURI   = "X-Forwarded-Uri"
	SamlRootURL     = "saml-root-url"
)

type Director struct{}

// GetRedirect determines the full URL or URI path to redirect clients to once
// authenticated with the OAuthProxy.
// Strategy priority (first legal result is used):
// - `rd` querysting parameter
// - `X-Auth-Request-Redirect` header
// - `X-Forwarded-(Proto|Host|Uri)` headers (when ReverseProxy mode is enabled)
// - `X-Forwarded-(Proto|Host)` if `Uri` has the ProxyPath (i.e. /oauth2/*)
// - `X-Forwarded-Uri` direct URI path (when ReverseProxy mode is enabled)
// - `req.URL.RequestURI` if not under the ProxyPath (i.e. /oauth2/*)
// - `/`
func (d Director) GetRedirect(req *http.Request) (string, error) {
	query := req.URL.Query()
	redirectUrl := query.Get("rd")
	return redirectUrl, nil
}

func (d Director) GetRootUrl(req *http.Request) (string, error) {
	protocol := req.Header.Get(XForwardedProto)
	if protocol == "" {
		return "", errors.New(XForwardedProto + " header is missing")
	}
	host := req.Header.Get(XForwardedHost)
	if host == "" {
		return "", errors.New(XForwardedHost + " header is missing")
	}
	rootUrl := fmt.Sprintf("%s://%s", protocol, host)
	return rootUrl, nil
}

func (d Director) GetRootUrlFromGinContext(context *gin.Context) (string, error) {
	rawRootUrl, exists := context.Get(SamlRootURL)
	if !exists {
		return "", errors.New("root url is missing")
	}

	rootUrl := rawRootUrl.(string)
	parse, err := url.Parse(rootUrl)
	return fmt.Sprintf("%s://%s", parse.Scheme, parse.Host), err
}
