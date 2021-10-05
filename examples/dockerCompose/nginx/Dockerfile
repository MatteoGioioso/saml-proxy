FROM public.ecr.aws/bitnami/nginx:1.20

COPY nginx.conf /opt/bitnami/nginx/conf/server_blocks/nginx.conf
COPY localhost/cert.pem /opt/bitnami/nginx/conf/bitnami/certs/cert.pem
COPY localhost/key.pem /opt/bitnami/nginx/conf/bitnami/certs/key.pem
USER root

EXPOSE 8080