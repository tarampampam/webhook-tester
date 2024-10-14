# syntax=docker/dockerfile:1

# -✂- this stage is used to develop and build the application locally -------------------------------------------------
FROM docker.io/library/node:22-bookworm AS develop

# install Go using the official image
COPY --from=docker.io/library/golang:1.23-bookworm /usr/local/go /usr/local/go

ENV \
  # add Go and Node.js "binaries" to the PATH
  PATH="$PATH:/src/web/node_modules/.bin:/go/bin:/usr/local/go/bin" \
  # use the /var/tmp/go as the GOPATH to reuse the modules cache
  GOPATH="/var/tmp/go" \
  # set path to the Go cache (think about this as a "object files cache")
  GOCACHE="/var/tmp/go/cache" \
  # disable npm update notifier
  NPM_CONFIG_UPDATE_NOTIFIER=false

# install development tools and dependencies
RUN set -x \
    # renovate: source=github-releases name=oapi-codegen/oapi-codegen
    && OAPI_CODEGEN_VERSION="2.4.1" \
    && GOBIN=/bin go install "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v${OAPI_CODEGEN_VERSION}" \
    && go clean -cache -modcache \
    # renovate: source=github-releases name=golangci/golangci-lint
    && GOLANGCI_LINT_VERSION="1.61.0" \
    && wget -O- -nv "https://cdn.jsdelivr.net/gh/golangci/golangci-lint@v${GOLANGCI_LINT_VERSION}/install.sh" \
      | sh -s -- -b /bin "v${GOLANGCI_LINT_VERSION}" \
    # customize the shell prompt (for the bash)
    && echo "PS1='\[\033[1;36m\][develop] \[\033[1;34m\]\w\[\033[0;35m\] \[\033[1;36m\]# \[\033[0m\]'" >> /etc/bash.bashrc

WORKDIR /src

RUN \
    --mount=type=bind,source=web/package.json,target=/src/web/package.json \
    --mount=type=bind,source=web/package-lock.json,target=/src/web/package-lock.json \
    --mount=type=bind,source=go.mod,target=/src/go.mod \
    --mount=type=bind,source=go.sum,target=/src/go.sum \
    set -x \
    # install node dependencies
    && npm --prefix /src/web ci -dd --no-audit --prefer-offline \
    # burn the Go modules cache
    && go mod download -x \
    # allow anyone to read/write the Go cache
    && find /var/tmp/go -type d -exec chmod 0777 {} + \
    && find /var/tmp/go -type f -exec chmod 0666 {} +

# -✂- this stage is used to build the application frontend ------------------------------------------------------------
FROM develop AS frontend

# copy the frontend source code
COPY ./web /src/web

# build the frontend (built artifact can be found in /src/web/dist)
RUN --mount=type=bind,source=api/openapi.yml,target=/src/api/openapi.yml \
    set -x \
    && npm --prefix /src/web run generate \
    && npm --prefix /src/web run build

# -✂- this stage is used to compile the application -------------------------------------------------------------------
FROM develop AS compile

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
      -ldflags "-s -w -X gh.tarampamp.am/webhook-tester/internal/version.version=${APP_VERSION}" \
      -o ./app \
      ./cmd/app/ \
    && ./app --version \
    # prepare rootfs for runtime
    && mkdir -p /tmp/rootfs \
    && cd /tmp/rootfs \
    && mkdir -p ./etc/ssl/certs ./bin ./tmp \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && chmod 777 ./tmp \
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
COPY --from=compile /tmp/rootfs /

# use an unprivileged user
USER 10001:10001

ENV \
  # logging format
  LOG_FORMAT=json \
  # logging level
  LOG_LEVEL=info

#EXPOSE "80/tcp" "443/tcp"

HEALTHCHECK --interval=10s --start-interval=1s --start-period=5s --timeout=1s CMD ["/bin/app", "http-server", "healthcheck"]
ENTRYPOINT ["/bin/app"]
CMD ["start"]
