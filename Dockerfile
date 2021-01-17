# syntax=docker/dockerfile:1.2

# Image page: <https://hub.docker.com/_/golang>
FROM --platform=${TARGETPLATFORM:-linux/amd64} golang:1.15.6-alpine as builder

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

COPY . /src

# arguments to pass on each go tool link invocation
ENV LDFLAGS="-s -w -X webhook-tester/version.version=$APP_VERSION"

RUN set -x \
    && go version \
    && CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o /tmp/webhook-tester . \
    && /tmp/webhook-tester version \
    && /tmp/webhook-tester -h

# prepare rootfs for runtime
RUN set -x \
    && mkdir -p /tmp/rootfs/etc/ssl /tmp/rootfs/etc/apache2 \
    && mkdir -p /tmp/rootfs/bin \
    && mkdir -p /tmp/rootfs/opt/webhook-tester \
    && cp -R /etc/ssl/certs /tmp/rootfs/etc/ssl/certs \
    && cp /etc/mime.types /tmp/rootfs/etc/mime.types \
    && cp /etc/apache2/mime.types /tmp/rootfs/etc/apache2/mime.types \
    && cp -R /src/public /tmp/rootfs/opt/webhook-tester/public \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > /tmp/rootfs/etc/passwd \
    && echo 'appuser:x:10001:' > /tmp/rootfs/etc/group \
    && mv /tmp/webhook-tester /tmp/rootfs/bin/webhook-tester

# use empty filesystem
FROM scratch

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

# Import from builder
COPY --from=builder /tmp/rootfs /

# Use an unprivileged user
USER appuser:appuser

# Docs: <https://docs.docker.com/engine/reference/builder/#healthcheck>
# TODO add healthcheck
#HEALTHCHECK --interval=15s --timeout=3s --start-period=1s CMD [ \
#    "/bin/webhook-tester", "healthcheck", \
#    "--log-json", \
#    "--port", "8080" \
#]

ENTRYPOINT ["/bin/webhook-tester"]

# TODO append "--log-json" flag
CMD [ \
    "serve", \
    "--port", "8080", \
    "--public", "/opt/webhook-tester/public" \
]
