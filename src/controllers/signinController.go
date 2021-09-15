package controllers

import (
	"encoding/base64"
	"fmt"
	"github.com/MatteoGioioso/saml-proxy/director"
	"github.com/MatteoGioioso/saml-proxy/domain"
	"github.com/MatteoGioioso/saml-proxy/sharedKernel"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type SigninController struct {
	Router     *gin.Engine
	SamlDomain *domain.SamlDomain
	Logger     sharedKernel.Logger
	Director   director.Director
	rootUrl    string
	middleware *samlsp.Middleware
}

func (c *SigninController) Handler() gin.IRoutes {
	return c.Router.GET("/saml/sign_in", func(context *gin.Context) {
		rootUrl, err := c.Director.GetRootUrl(context.Request)
		fmt.Println("Signin controller root url: ", rootUrl)
		if err != nil {
			context.Writer.WriteHeader(500)
			context.Writer.Write([]byte(err.Error()))
			return
		}
		c.rootUrl = rootUrl
		c.middleware = c.SamlDomain.GetProvider(c.rootUrl)
		c.handleStartAuthFlow(context.Writer, context.Request)
	})
}

func (c SigninController) handleStartAuthFlow(w http.ResponseWriter, r *http.Request) {
	// If we try to redirect when the original request is the ACS URL we'll
	// end up in a loop. This is a programming error, so we panic here. In
	// general this means a 500 to the user, which is preferable to a
	// redirect loop.
	if r.URL.Path == c.middleware.ServiceProvider.AcsURL.Path {
		panic("don't wrap Middleware with RequireAccount")
	}

	var binding, bindingLocation string
	if c.middleware.Binding != "" {
		binding = c.middleware.Binding
		bindingLocation = c.middleware.ServiceProvider.GetSSOBindingLocation(binding)
	} else {
		binding = saml.HTTPRedirectBinding
		bindingLocation = c.middleware.ServiceProvider.GetSSOBindingLocation(binding)
		if bindingLocation == "" {
			binding = saml.HTTPPostBinding
			bindingLocation = c.middleware.ServiceProvider.GetSSOBindingLocation(binding)
		}
	}

	authReq, err := c.middleware.ServiceProvider.MakeAuthenticationRequest(bindingLocation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// relayState is limited to 80 bytes but also must be integrity protected.
	// this means that we cannot use a JWT because it is way to long. Instead
	// we set a signed cookie that encodes the original URL which we'll check
	// against the SAML response when we get it.
	relayState, err := c.trackRequest(w, r, authReq.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if binding == saml.HTTPRedirectBinding {
		redirectURL := authReq.Redirect(relayState)
		w.Header().Add("Location", redirectURL.String())
		w.WriteHeader(http.StatusFound)
		return
	}
	if binding == saml.HTTPPostBinding {
		w.Header().Add("Content-Security-Policy", ""+
			"default-src; "+
			"script-src 'sha256-AjPdJSbZmeWHnEc5ykvJFay8FTWeTeRbs9dutfZ0HqE='; "+
			"reflected-xss block; referrer no-referrer;")
		w.Header().Add("Content-type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>`))
		w.Write(authReq.Post(relayState))
		w.Write([]byte(`</body></html>`))
		return
	}
	panic("not reached")
}

func (c SigninController) trackRequest(w http.ResponseWriter, r *http.Request, samlRequestID string) (string, error) {
	cookieRequestTracker := c.middleware.RequestTracker.(samlsp.CookieRequestTracker)
	redirect, err := c.Director.GetRedirect(r)
	if err != nil {
		return "", err
	}

	trackedRequest := samlsp.TrackedRequest{
		Index:         base64.RawURLEncoding.EncodeToString(randomBytes(42)),
		SAMLRequestID: samlRequestID,
		URI:           redirect,
	}
	signedTrackedRequest, err := cookieRequestTracker.Codec.Encode(trackedRequest)
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieRequestTracker.NamePrefix + trackedRequest.Index,
		Value:    signedTrackedRequest,
		MaxAge:   int(cookieRequestTracker.MaxAge.Seconds()),
		HttpOnly: true,
		SameSite: cookieRequestTracker.SameSite,
		Secure:   cookieRequestTracker.ServiceProvider.AcsURL.Scheme == "https",
		Path:     cookieRequestTracker.ServiceProvider.AcsURL.Path,
	})

	return trackedRequest.Index, nil
}

func randomBytes(n int) []byte {
	rv := make([]byte, n)

	if _, err := io.ReadFull(saml.RandReader, rv); err != nil {
		panic(err)
	}
	return rv
}
