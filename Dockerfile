FROM public.ecr.aws/bitnami/golang:1.16 AS builder

COPY src/go.mod src/go.sum ./
RUN unset GOPATH && go mod tidy && go mod download

FROM builder AS builder-02

COPY src/. .
RUN unset GOPATH && CGO_ENABLED=0 go build -o bin/main .

FROM public.ecr.aws/micahhausler/alpine:3.14.0
RUN apk -U upgrade
RUN addgroup -S saml-proxy --gid 1000 && adduser -S saml-proxy --uid 1000 -G saml-proxy

COPY --from=builder-02 /go/bin/main /saml-proxy/main
RUN chown -R saml-proxy:saml-proxy /saml-proxy

USER 1000

EXPOSE 9000

WORKDIR /saml-proxy
ENTRYPOINT ["./main"]
