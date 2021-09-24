#!/usr/bin/env sh

export SAML_PROXY_METADATA_ENDPOINT=https://myidp.com/metadata/000000000000
export SAML_PROXY_HOSTS='["dashboard.mycoolsaml.com"]'
export SAML_PROXY_ALLOW_IDP_INITIATED=true
export SAML_PROXY_SIGN_REQUEST=true
export SAML_PROXY_SSL_CERTIFICATE_PATH=/path/to/my/certs/cert.pem
export SAML_PROXY_SSL_CERTIFICATE_KEY_PATH=/path/to/my/certs/key.pem
export SAML_PROXY_SSL_CERTIFICATE_AUTOGENERATE=false
export SAML_PROXY_PROTOCOL=https
export PORT=9000
