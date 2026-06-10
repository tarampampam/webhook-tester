# syntax=docker/dockerfile:1

# -✂- this stage is used to develop and build the application locally -------------------------------------------------
FROM docker.io/library/golang:1.26.4-alpine AS builder

# path to the directory used by go to store cache and modules (/var/cache/go)
ENV GOCACHE=/var/cache/go/build GOMODCACHE=/var/cache/go/mod

# disable golang telemetry (https://go.dev/doc/telemetry#faq)
# https://go.googlesource.com/telemetry/+/refs/tags/config/v0.100.0/internal/telemetry/dir.go#90
RUN set -x \
    && mkdir -p ~/.config/go/telemetry \
    && echo "off $(date -u +%Y-%m-%d)" > ~/.config/go/telemetry/mode

# path to the directory used by npm and node to store cache
ENV NODE_COMPILE_CACHE=/var/cache/node

# install nodejs using the official image
RUN --mount=type=bind,from=docker.io/library/node:25-alpine,source=/,target=/mnt \
    set -x \
    && cp /mnt/usr/local/bin/node /usr/local/bin/ \
    && cp /mnt/usr/lib/libgcc_s.so.1 /mnt/usr/lib/libstdc++.so.6 /usr/lib/ \
    && cp -r /mnt/usr/local/lib/node_modules /usr/local/lib/ \
    && ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && cd /usr/local/lib \
    && rm -r ./node_modules/npm/docs ./node_modules/npm/man \
    && find ./node_modules/ -type f \( -name "*.map" -o -name "*.md" \) -delete \
    && npm config set --global "update-notifier=false" "loglevel=error" "cache=$NODE_COMPILE_CACHE" \
    && rm -r ~/.npm \
    && node --version && npm --version && npx --version \
    && rm -r "$NODE_COMPILE_CACHE" \
    && echo "default:x:1000:1000::/tmp:/bin/sh" >> /etc/passwd

# prepare cache directories with proper permissions
RUN set -x \
    && mkdir -p "$GOCACHE" "$GOMODCACHE" "$NODE_COMPILE_CACHE" \
    && find "$GOCACHE" "$GOMODCACHE" "$NODE_COMPILE_CACHE" -type d -exec chmod 777 {} \; \
    && find "$GOCACHE" "$GOMODCACHE" "$NODE_COMPILE_CACHE" -type f -exec chmod a+rwX {} \;

# install gcc to be able to run tests with race detector
RUN apk add --no-cache gcc musl-dev

# -✂- this stage is used to build the application frontend ------------------------------------------------------------
FROM builder AS frontend

# build the frontend (built artifact can be found in /src/web/dist)
RUN --mount=type=bind,source=api,target=/src/api \
    --mount=type=bind,source=web,target=/mnt/web \
    set -x \
    && cp -r /mnt/web /src/web \
    && cd /src/web \
    && npm install --loglevel http --no-audit --no-fund \
    && npm run generate \
    && npm run build \
    && mv ./dist /tmp/dist \
    && rm -r /src/web \
    && mkdir /src/web \
    && mv /tmp/dist /src/web/dist \
    && rm -r "$NODE_COMPILE_CACHE"

# -✂- this stage is used to build the app itself (including frontend embedding) ---------------------------------------
FROM builder AS backend

# can be passed with any prefix (like `v1.2.3@GITHASH`), e.g.: `(podman|docker) build --build-arg "APP_VERSION=v1.2.3" .`
ARG APP_VERSION="undefined@docker"

# build the application itself (built artifact can be found in /tmp/webhook-tester)
RUN --mount=type=bind,source=.,target=/mnt/src \
    --mount=type=bind,from=frontend,source=/src/web/dist,target=/mnt/dist \
    set -x \
    && cp -r /mnt/src /src \
    && cp -r /mnt/dist /src/web/dist \
    && cd /src \
    && go generate -skip readme ./... \
    && CGO_ENABLED=0 go build \
      -trimpath \
      -buildvcs=false \
      -ldflags "-s -w -X gh.tarampamp.am/webhook-tester/v3/internal/appmeta.version=${APP_VERSION}" \
      -o /tmp/webhook-tester \
      ./cmd/webhook-tester/ \
    && go clean -cache -modcache -i \
    && /tmp/webhook-tester --version \
    && rm -r /src

WORKDIR /tmp/rootfs

# prepare rootfs for runtime
RUN set -x \
    && mkdir -p ./etc/ssl/certs ./bin ./tmp ./data \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && chmod 777 ./tmp ./data \
    && cp /etc/ssl/certs/ca-certificates.crt ./etc/ssl/certs/ \
    && mv /tmp/webhook-tester ./bin/webhook-tester

# add super-lightweight HTTP checking tool to use it in the healthcheck
# docs: https://github.com/tarampampam/microcheck
COPY --from=ghcr.io/tarampampam/microcheck:1 /bin/httpscheck /tmp/rootfs/bin/httpscheck

# -✂- and this is the final stage -------------------------------------------------------------------------------------
FROM scratch AS runtime

ARG APP_VERSION="undefined@docker"

# docs: https://github.com/opencontainers/image-spec/blob/master/annotations.md
LABEL \
    org.opencontainers.image.title="webhook-tester" \
    org.opencontainers.image.description="Test your HTTP webhooks using friendly web UI" \
    org.opencontainers.image.url="https://github.com/tarampampam/webhook-tester" \
    org.opencontainers.image.source="https://github.com/tarampampam/webhook-tester" \
    org.opencontainers.image.vendor="tarampampam" \
    org.opencontainers.version="$APP_VERSION" \
    org.opencontainers.image.licenses="MIT"

# use an unprivileged user by dedault
USER 10001:10001

# import rootfs from the backend stage
COPY --from=backend /tmp/rootfs /

ENV LOG_FORMAT=json \
    LOG_LEVEL=info \
    FS_STORAGE_DIR=/data

# docs: https://docs.docker.com/reference/dockerfile/#healthcheck
HEALTHCHECK --interval=10s --start-interval=1s --start-period=1s CMD [\
  "/bin/httpscheck", "--port-env", "HTTP_PORT", "127.0.0.1:8080/ready"\
]

EXPOSE "8080/tcp"

ENTRYPOINT ["/bin/webhook-tester"]
