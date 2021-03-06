# Saml proxy

Small SAML 2.0 Service provider authentication proxy.
I mainly built this to leverage direct AWS SSO authentication with external services such Kubernetes, Linkerd, Grafana dashboards.

> **NOTE** this proxy is tested using Nginx and Nginx ingress controller only

## Usage

### Configuration with AWS SSO

To be used with AWS SSO you need to create one Application per host.
To create a custom application follow those the docs [here](https://docs.aws.amazon.com/singlesignon/latest/userguide/samlapps.html) or [here](https://static.global.sso.amazonaws.com/app-520727d4117d1647/instructions/index.htm?metadata=https)

One other important setting is the attributes mapping, you need to set the `Subject` to `transient` as showed below:

![attribute mappings](assets/aws_sso_attribute_mappings.png)


### Kubernetes

You can deploy Saml-proxy using the helm chart:

```shell
 helm repo add saml-proxy https://matteogioioso.github.io/saml-proxy/
 helm repo update
```

You can use this values and use one host per dashboard:
```yaml
config:
  samlMetadataEndpoint: "https://portal.sso.ap-southeast-1.amazonaws.com/saml/metadata/000xxxxxxxXxxxx0000000"
  samlHosts: [linkerd.company.com]

ingress:
  enabled: true
  className: "nginx"
  annotations: {}
  hosts:
    - host: linkerd.company.com
      paths:
        - path: /saml
          pathType: Prefix
```

With Nginx ingress controller:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: linkerd-dashboard-ingress
  namespace: linkerd-viz
  annotations:
    nginx.ingress.kubernetes.io/upstream-vhost: $service_name.$namespace.svc.cluster.local:8084
    nginx.ingress.kubernetes.io/configuration-snippet: |
      proxy_set_header Origin "";
      proxy_hide_header l5d-remote-ip;
      proxy_hide_header l5d-server-id;
    nginx.ingress.kubernetes.io/auth-url: "https://linkerd.company.com/saml/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://linkerd.company.com/saml/sign_in?rd=$host$request_uri"
    nginx.ingress.kubernetes.io/proxy-buffer-size: "8k"
    nginx.ingress.kubernetes.io/proxy-buffering: "on"
spec:
  ingressClassName: nginx
  rules:
    - host: linkerd.company.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 8084
```

### Docker and nginx

Your Nginx config should look something like this:

```
server {
        ...
        server_name localhost;

        location /saml/ {
	      proxy_pass              http://saml-proxy:9000;
	      proxy_set_header        Host $host;
	      proxy_set_header        X-Auth-Request-Redirect $request_uri;
          proxy_set_header        X-Forwarded-Uri $request_uri;
          proxy_set_header        X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header        X-Forwarded-Proto $scheme;
          proxy_set_header        X-Forwarded-Host $host;
        }

        location = /saml/auth {
          internal;
          proxy_pass              http://saml-proxy:9000;
          proxy_pass_request_body off;
          proxy_set_header        Content-Length "";
          proxy_set_header        X-Original-URI $request_uri;
          proxy_set_header        X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header        X-Forwarded-Proto $scheme;
          proxy_set_header        X-Forwarded-Host $host;
        }

        location / {
		    auth_request /saml/auth;
			error_page 401 = /saml/sign_in?rd=$host$request_uri;

		    auth_request_set $auth_cookie $upstream_http_set_cookie;
		    add_header Set-Cookie $auth_cookie;

		   proxy_buffer_size          256k;
           proxy_buffers              4 512k;
           proxy_busy_buffers_size    512k;

		   proxy_pass http://dashboard:5000;
	    }
}

```

And then your `docker-compose.yaml`:

```yaml
services:
  proxy:
    build:
      context: nginx/
      dockerfile: Dockerfile
    ports:
      - "443:443"
    networks:
      - saml-proxy-network

  dashboard:
    networks:
      - saml-proxy-network
    build:
      context: dashboard
      dockerfile: Dockerfile

  saml-proxy:
    image: public.ecr.aws/hirvitek/saml-proxy:latest
    networks:
      - saml-proxy-network
    environment:
      - SAML_PROXY_METADATA_ENDPOINT=https://my-idp/metadata/xxxxxxxxxxxxxxxxxx
      - SAML_PROXY_HOSTS=["mydashboard.exampl.com"]
      - SAML_PROXY_SSL_CERTIFICATE_AUTOGENERATE=true
      - PORT=9000
```

You can check the full example and run it locally in the example folder: `example/dockerCompose`

---

## Config 

| Environmental variable                    	| Helm variable                       	| Type                  	| Description                                                                                                                                                                                              	| Default 	| Example 	|
|-------------------------------------------	|-------------------------------------	|-----------------------	|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------	|---------	|---------	|
|  `SAML_PROXY_METADATA_ENDPOINT`           	| `config.samlMetadataEndpoint`       	| string                	| The metadata endpoint of your Identity provider                                                                                                                                                          	| ""      	| https://portal.sso.ap-southeast-1.amazonaws.com/saml/metadata/000xxxxxxxXxxxx0000000         	|
| `SAML_PROXY_HOSTS`                        	| `config.samlHosts`                  	| JSON array of strings 	| List of allowed hosts                                                                                                                                                                                    	| []      	| [linkerd.company.com, grafana.company.com]        	|
| `SAML_PROXY_ENTITY_ID`                      | `config.samlEntityId`               	| string                  | The identity provider entity id                                                                                                                                                                          	| ""       	| SAMLProxy        	|
| `SAML_PROXY_ALLOW_IDP_INITIATED`          	| `config.samlAllowIdpInitiated`      	| boolean               	| Allow authentication directly from the identity provider                                                                                                                                                 	| true    	|         	|
| `SAML_PROXY_SIGN_REQUEST`                 	| `config.samlSignRequest`            	| boolean               	| Sign the SAML request using the certificates                                                                                                                                                             	| true    	|         	|
| `SAML_PROXY_SSL_CERTIFICATE_PATH`         	| `config.sslCertificatePath`         	| string                	| If you decide to bring your own TLS certificates you can specify the path here (Note: you don't need to use this if `SAML_PROXY_SSL_CERTIFICATE_AUTOGENERATE` is set to true)                            	| ""      	| /path/to/certs/cert.crt       	|
| `SAML_PROXY_SSL_CERTIFICATE_KEY_PATH`     	| `config.sslCertificateKeyPath`      	| string                	| If you decide to bring your own TLS certificates you can specify the path of the certificate's key here (Note: you don't need to use this if  `SAML_PROXY_SSL_CERTIFICATE_AUTOGENERATE`  is set to true) 	| ""      	| /path/to/certs/cert.key        	|
| `SAML_PROXY_SSL_CERTIFICATE_AUTOGENERATE` 	| `config.sslCertificateAutogenerate` 	| boolean               	| If set to true it will auto-generate self-signed certificates everytime the server starts, set this to false if you are using custom TLS                                                                  | true    	|         	|
| `SAML_PROXY_PROTOCOL`                     	| `config.protocol`                   	| string                	| Useful if you want to test the proxy locally using                                                                                                                                                       	| "https" 	|         	|
| `PORT`                                    	| `config.proxyPort`                  	| number                	| The proxy server port                                                                                                                                                                                    	| 9000    	|         	|