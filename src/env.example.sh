#!/usr/bin/env sh

export SAML_METADATA_ENDPOINT=https://myidp.com/metadata/000000000000
export SAML_HOSTS='["https://dashboard.mycoolsaml.com"]'
export PORT=9000
export SSL_CERTIFICATE_PATH=/path/to/my/certs/cert.pem
export SSL_CERTIFICATE_KEY_PATH=/path/to/my/certs/key.pem