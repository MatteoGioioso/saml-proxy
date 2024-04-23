FROM golang:1.20 AS builder

COPY src/go.mod src/go.sum ./
RUN unset GOPATH && go mod tidy && go mod download

FROM builder AS builder-02

COPY src/. .
RUN unset GOPATH && CGO_ENABLED=0 go build -o bin/main .

FROM ubuntu:20.04 AS final

ARG GIN_MODE=release
ARG USER=saml-proxy
ARG GROUP=saml
ARG UID=1001
ARG GID=1001

ENV GIN_MODE=$GIN_MODE
ENV USER=$USER
ENV GROUP=$GROUP
ENV UID=$UID
ENV GID=$GID

RUN DEBIAN_FRONTEND=noninteractive \
    && apt-get update && apt-get upgrade -y

RUN addgroup $GROUP
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "$(pwd)" \
    --ingroup "$GROUP" \
    --no-create-home \
    --uid "$UID" \
    "$USER"


COPY --from=builder-02 /go/bin/main /saml-proxy/main
RUN chown -R $GID:$UID /saml-proxy

USER $USER

EXPOSE 9000

WORKDIR /saml-proxy
ENTRYPOINT ["./main"]
