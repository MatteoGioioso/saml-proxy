build:
	go mod tidy && go build -o saml-proxy server.go

run: build
	. ./env.sh && ./saml-proxy