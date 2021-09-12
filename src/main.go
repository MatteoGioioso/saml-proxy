package main

import (
	"fmt"
	"github.com/crewjam/saml/samlsp"
	"log"
	"net/http"
	"os"
)

func mainHandler(samlSP RouteGenerator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", samlsp.AttributeFromContext(r.Context(), "cn"))
	}
}

func genericHandler(generator RouteGenerator, app http.HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		samlSP, err := generator.CreateSamlService(r.Host)
		if err != nil {
			log.Fatal(err)
		}

		if r.URL.Path == "/saml/acs" {
			samlSP.ServeHTTP(w, r)
			return
		}

		account := samlSP.RequireAccount(app)
		account.ServeHTTP(w, r)
	}
}

func main() {
	generator := RouteGenerator{
		MetadataEndpoint: os.Getenv("SAML_METADATA_ENDPOINT"),
	}

	app := http.HandlerFunc(mainHandler(generator))
	handler := http.HandlerFunc(genericHandler(generator, app))

	if err := http.ListenAndServe(":9000", handler); err != nil {
		log.Fatal(err)
	}
}
