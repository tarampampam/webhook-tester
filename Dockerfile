# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.15-alpine as builder

# can be passed with any prefix (like `v1.2.3@GITHASH`)
# e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@GITHASH" .`
ARG APP_VERSION="undefined@docker"

RUN set -x \
    && mkdir /src \
    # SSL ca certificates (ca-certificates is required to call HTTPS endpoints)
    # packages mailcap and apache2 is needed for /etc/mime.types and /etc/apache2/mime.types files respectively
    && apk add --no-cache mailcap apache2 ca-certificates \
    && update-ca-certificates

WORKDIR /src

COPY ./go.mod ./go.sum ./

# Burn modules cache
RUN set -x \
    && go version \
    && go mod download \
    && go mod verify

COPY . /src

RUN set -x \
    && go version \
    && GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X webhook-tester/version.version=${APP_VERSION}" -o /tmp/webhook-tester . \
    && /tmp/webhook-tester version \
    && /tmp/webhook-tester -h

# Image page: <https://hub.docker.com/_/alpine>
FROM alpine:latest as runtime

ARG APP_VERSION="undefined@docker"

LABEL \
    # Docs: <https://github.com/opencontainers/image-spec/blob/master/annotations.md>
    org.opencontainers.image.title="webhook-tester" \
    org.opencontainers.image.description="Test your HTTP webhooks using friendly web UI" \
    org.opencontainers.image.url="https://github.com/tarampampam/webhook-tester" \
    org.opencontainers.image.source="https://github.com/tarampampam/webhook-tester" \
    org.opencontainers.image.vendor="tarampampam" \
    org.opencontainers.version="$APP_VERSION" \
    org.opencontainers.image.licenses="MIT"

RUN set -x \
    # Unprivileged user creation <https://stackoverflow.com/a/55757473/12429735RUN>
    && adduser \
        --disabled-password \
        --gecos "" \
        --home /nonexistent \
        --shell /sbin/nologin \
        --no-create-home \
        --uid 10001 \
        appuser

# Use an unprivileged user
USER appuser:appuser

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/webhook-tester /app/webhook-tester
COPY --from=builder /etc/mime.types /etc/mime.types
COPY --from=builder /etc/apache2/mime.types /etc/apache2/mime.types
COPY --chown=appuser ./public /app/public

WORKDIR /app

ENTRYPOINT ["/app/webhook-tester"]

CMD ["serve"]
