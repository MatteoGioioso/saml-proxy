package main

import (
	"fmt"
	"github.com/crewjam/saml/samlsp"
	"log"
	"net/http"
	"net/url"
	"os"
)

func genericHandler(generator RouteGenerator, samlSP *samlsp.Middleware) func(w http.ResponseWriter, r *http.Request) {
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
			u := &url.URL{
				Scheme:      "http",
				Opaque:      "",
				User:        nil,
				Host:        "mycoolsaml:8080",
				Path:        "/",
				RawPath:     "",
				ForceQuery:  false,
				RawQuery:    "",
				Fragment:    "",
				RawFragment: "",
			}
			r.URL = u
			samlSP.HandleStartAuthFlow(w, r)
			return
		}

		if r.URL.Path == "/saml/acs" {
			trackedRequestIndex := r.Form.Get("RelayState")
			fmt.Println(trackedRequestIndex)
			samlSP.ServeHTTP(w, r)
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

	samlSP, err := generator.CreateSamlService("localhost:9000")
	if err != nil {
		log.Fatal(err)
	}

	//app := http.HandlerFunc(mainHandler(generator))
	handler := http.HandlerFunc(genericHandler(generator, samlSP))

	if err := http.ListenAndServe(":9000", handler); err != nil {
		log.Fatal(err)
	}
}
