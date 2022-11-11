# syntax=docker/dockerfile:1.2

# Image page: <https://hub.docker.com/_/node>
FROM node:19-alpine as frontend

COPY . /src

WORKDIR /src/web

# install node dependencies
RUN set -x \
    && npm config set update-notifier false \
    && npm ci --no-audit --prefer-offline

# build the frontend (built artifact can be found in /src/web/dist)
RUN set -x \
    && npm run generate \
    && npm run build

# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.19-alpine as builder

# can be passed with any prefix (like `v1.2.3@GITHASH`)
# e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@GITHASH" .`
ARG APP_VERSION="undefined@docker"

COPY . /src

WORKDIR /src

COPY --from=frontend /src/web/dist /src/web/dist

# arguments to pass on each go tool link invocation
ENV LDFLAGS="-s -w -X github.com/tarampampam/webhook-tester/internal/pkg/version.version=$APP_VERSION"

RUN set -x \
    && go generate ./... \
    && CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o /tmp/webhook-tester ./cmd/webhook-tester/ \
    && /tmp/webhook-tester version \
    && /tmp/webhook-tester -h

# prepare rootfs for runtime
RUN mkdir -p /tmp/rootfs

WORKDIR /tmp/rootfs

RUN set -x \
    && mkdir -p \
        ./etc \
        ./bin \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && mv /tmp/webhook-tester ./bin/webhook-tester

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
HEALTHCHECK --interval=15s --timeout=3s --start-period=1s CMD [ \
    "/bin/webhook-tester", "healthcheck", \
    "--log-json", \
    "--port", "8080" \
]

ENTRYPOINT ["/bin/webhook-tester"]

CMD ["serve", "--log-json"]
