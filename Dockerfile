# syntax=docker/dockerfile:1

# -✂- this stage is used to develop and build the application locally -------------------------------------------------
FROM docker.io/library/node:24-alpine AS builder

# install Go using the official image
COPY --from=docker.io/library/golang:1.25-alpine /usr/local/go /usr/local/go

ENV \
  # add Go and Node.js "binaries" to the PATH
  PATH="$PATH:/src/web/node_modules/.bin:/usr/local/go/bin" \
  # tell the Go command where to find (and store) various stuff (instead of ~/go)
  GOPATH="/var/tmp/go" \
  # disable npm update notifier
  NPM_CONFIG_UPDATE_NOTIFIER=false

WORKDIR /src

# burn the dependencies cache
RUN --mount=type=bind,source=go.mod,target=/src/go.mod \
    --mount=type=bind,source=go.sum,target=/src/go.sum \
    --mount=type=bind,source=tools.go.mod,target=/src/tools.go.mod \
    --mount=type=bind,source=tools.go.sum,target=/src/tools.go.sum \
    --mount=type=bind,source=web/package.json,target=/src/web/package.json \
    --mount=type=bind,source=web/package-lock.json,target=/src/web/package-lock.json \
    set -x \
    # for the Go modules
    && mkdir -p "$GOPATH" \
    && go mod download -x \
    && go mod download -modfile=tools.go.mod -x \
    # and for the Node.js packages
    && npm --prefix /src/web ci --loglevel verbose --no-audit \
    # allow read/write for everyone to use the cache from any user (including non-root)
    && find "$GOPATH" /src/web/node_modules -type d -exec chmod 0777 {} + \
    && find "$GOPATH" /src/web/node_modules -type f -exec chmod a+rwX {} +

ENTRYPOINT [""]

# -✂- this stage is used to build the application frontend ------------------------------------------------------------
FROM builder AS frontend

# copy the frontend source code
COPY ./web /src/web

# build the frontend (built artifact can be found in /src/web/dist)
RUN --mount=type=bind,source=api/openapi.yml,target=/src/api/openapi.yml \
    set -x \
    && npm --prefix /src/web run generate \
    && npm --prefix /src/web run build

# -✂- this stage is used to build the app itself (including frontend embedding) ---------------------------------------
FROM builder AS backend

# can be passed with any prefix (like `v1.2.3@FOO`), e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@FOO" .`
ARG APP_VERSION="undefined@docker"

# copy the source code
COPY . /src

RUN --mount=type=bind,from=frontend,source=/src/web/dist,target=/src/web/dist \
    set -x \
    # build the app itself
    && go generate -skip readme ./... \
    && CGO_ENABLED=0 go build \
      -trimpath \
      -buildvcs=false \
      -ldflags "-s -w -X gh.tarampamp.am/webhook-tester/v2/internal/version.version=${APP_VERSION}" \
      -o ./app \
      ./cmd/webhook-tester/ \
    && ./app --version \
    # prepare rootfs for runtime
    && mkdir -p /tmp/rootfs \
    && cd /tmp/rootfs \
    && mkdir -p ./etc/ssl/certs ./bin ./tmp ./data \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && chmod 777 ./tmp ./data \
    && cp /etc/ssl/certs/ca-certificates.crt ./etc/ssl/certs/ \
    && mv /src/app ./bin/app

# -✂- and this is the final stage -------------------------------------------------------------------------------------
FROM scratch AS runtime

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

# import compiled application
COPY --from=backend /tmp/rootfs /

# use an unprivileged user
USER 10001:10001

ENV \
  # logging format
  LOG_FORMAT=json \
  # logging level
  LOG_LEVEL=info \
  # default fs storage directory
  FS_STORAGE_DIR=/data

#EXPOSE "8080/tcp"

HEALTHCHECK --interval=10s --start-interval=1s --start-period=5s --timeout=1s CMD ["/bin/app", "start", "healthcheck"]
ENTRYPOINT ["/bin/app"]
CMD ["start"]
