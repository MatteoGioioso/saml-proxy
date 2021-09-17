docker-build:
	docker build -t saml-proxy .

docker: docker-build
	. ./src/env.sh && docker run --rm -p "9000:9000" \
					-e SAML_METADATA_ENDPOINT=$${SAML_METADATA_ENDPOINT} \
					-e SAML_HOSTS=$${SAML_HOSTS} \
					-e SSL_CERTIFICATE_AUTOGENERATE=$${SSL_CERTIFICATE_AUTOGENERATE} \
					-e PORT=$${PORT} \
					saml-proxy