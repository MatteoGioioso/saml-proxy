FROM public.ecr.aws/bitnami/nginx:1.20

COPY nginx.conf /opt/bitnami/nginx/conf/server_blocks/nginx.conf
COPY index.html /app/index.html
COPY _.mycoolsaml.com/cert.pem /opt/bitnami/nginx/conf/bitnami/certs/cert.pem
COPY _.mycoolsaml.com/key.pem /opt/bitnami/nginx/conf/bitnami/certs/key.pem
USER root

EXPOSE 8080