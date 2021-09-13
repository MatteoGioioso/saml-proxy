build:
	go build -o saml-proxy main.go routes_generator.go

run: build
	. ./env.sh && ./saml-proxy