package main

import (
	"encoding/base64"
	"fmt"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"io"
	"log"
	"net/http"
	"os"
)


func randomBytes(n int) []byte {
	rv := make([]byte, n)

	if _, err := io.ReadFull(saml.RandReader, rv); err != nil {
		panic(err)
	}
	return rv
}

func serveACS(m *samlsp.Middleware, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	possibleRequestIDs := []string{}
	if m.ServiceProvider.AllowIDPInitiated {
		possibleRequestIDs = append(possibleRequestIDs, "")
	}

	trackedRequests := m.RequestTracker.GetTrackedRequests(r)
	for _, tr := range trackedRequests {
		possibleRequestIDs = append(possibleRequestIDs, tr.SAMLRequestID)
	}

	assertion, err := m.ServiceProvider.ParseResponse(r, possibleRequestIDs)
	if err != nil {
		m.OnError(w, r, err)
		return
	}

	createSessionFromAssertion(m, w, r, assertion)
	return
}

func createSession(m *samlsp.Middleware, w http.ResponseWriter, r *http.Request, assertion *saml.Assertion) error {
	c := m.Session.(samlsp.CookieSessionProvider)
	session, err := c.Codec.New(assertion)
	if err != nil {
		return err
	}

	value, err := c.Codec.Encode(session)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Domain:   ".mycoolsaml.com",
		Value:    value,
		MaxAge:   int(c.MaxAge.Seconds()),
		HttpOnly: c.HTTPOnly,
		Secure:   c.Secure || r.URL.Scheme == "https",
		SameSite: c.SameSite,
		Path:     "/",
	})
	return nil
}


func createSessionFromAssertion(m *samlsp.Middleware, w http.ResponseWriter, r *http.Request, assertion *saml.Assertion) {
	redirectURI := "/"
	if trackedRequestIndex := r.Form.Get("RelayState"); trackedRequestIndex != "" {
		trackedRequest, err := m.RequestTracker.GetTrackedRequest(r, trackedRequestIndex)
		if err != nil {
			m.OnError(w, r, err)
			return
		}
		m.RequestTracker.StopTrackingRequest(w, r, trackedRequestIndex)

		redirectURI = trackedRequest.URI
	}

	if err := createSession(m, w, r, assertion); err != nil {
		m.OnError(w, r, err)
		return
	}

	http.Redirect(w, r, redirectURI, http.StatusFound)
}


func trackRequest(m *samlsp.Middleware ,w http.ResponseWriter, r *http.Request, samlRequestID string) (string, error) {
	cookieRequestTracker := m.RequestTracker.(samlsp.CookieRequestTracker)
	trackedRequest := samlsp.TrackedRequest{
		Index:         base64.RawURLEncoding.EncodeToString(randomBytes(42)),
		SAMLRequestID: samlRequestID,
		URI:           "http://dashboard.mycoolsaml.com:8080",
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
		Domain: ".mycoolsaml.com",
	})

	return trackedRequest.Index, nil
}

func handleStartAuthFlow(m  *samlsp.Middleware,w http.ResponseWriter, r *http.Request) {
	// If we try to redirect when the original request is the ACS URL we'll
	// end up in a loop. This is a programming error, so we panic here. In
	// general this means a 500 to the user, which is preferable to a
	// redirect loop.
	if r.URL.Path == m.ServiceProvider.AcsURL.Path {
		panic("don't wrap Middleware with RequireAccount")
	}

	var binding, bindingLocation string
	if m.Binding != "" {
		binding = m.Binding
		bindingLocation = m.ServiceProvider.GetSSOBindingLocation(binding)
	} else {
		binding = saml.HTTPRedirectBinding
		bindingLocation = m.ServiceProvider.GetSSOBindingLocation(binding)
		if bindingLocation == "" {
			binding = saml.HTTPPostBinding
			bindingLocation = m.ServiceProvider.GetSSOBindingLocation(binding)
		}
	}

	authReq, err := m.ServiceProvider.MakeAuthenticationRequest(bindingLocation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// relayState is limited to 80 bytes but also must be integrity protected.
	// this means that we cannot use a JWT because it is way to long. Instead
	// we set a signed cookie that encodes the original URL which we'll check
	// against the SAML response when we get it.
	relayState, err := trackRequest(m, w, r, authReq.ID)
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

func genericHandler(samlSP *samlsp.Middleware) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.String())
		if r.URL.Path == "/saml/auth" {
			fmt.Printf("%+v\n", r.Header)
			session, err := samlSP.Session.GetSession(r)
			if session != nil {
				w.WriteHeader(200)
				return
			}

			if err == samlsp.ErrNoSession {
				w.WriteHeader(401)
				return
			}

			return
		}

		if r.URL.Path == "/saml/login" {
			fmt.Printf("%+v\n", r.Header)
			fmt.Printf("%+v\n", r.Host)
			handleStartAuthFlow(samlSP, w, r)
			return
		}

		if r.URL.Path == "/saml/acs" {
			serveACS(samlSP, w, r)
			return
		}

		if r.URL.Path == "/saml/metadata" {
			samlSP.ServeHTTP(w, r)
			return
		}
	}
}

func main() {
	generator := RouteGenerator{
		MetadataEndpoint: os.Getenv("SAML_METADATA_ENDPOINT"),
	}

	samlSP, err := generator.CreateSamlService("auth.mycoolsaml.com:9000")
	if err != nil {
		log.Fatal(err)
	}

	handler := http.HandlerFunc(genericHandler(samlSP))

	if err := http.ListenAndServe(":9000", handler); err != nil {
		log.Fatal(err)
	}
}
