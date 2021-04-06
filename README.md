<p align="center">
  <img src="https://hsto.org/webt/mn/fz/q-/mnfzq-lgdnbmv-3xv-1qm6gn82e.png" alt="Logo" width=128" />
</p>

# WebHook Tester

[![Release version][badge_release_version]][link_releases]
![Project language][badge_language]
[![Build Status][badge_build]][link_build]
[![Release Status][badge_release]][link_build]
[![Coverage][badge_coverage]][link_coverage]
[![License][badge_license]][link_license]

With this application you instantly get a unique, random URL that you can use to test and debug Webhooks and HTTP requests.

<p align="center">
  <img src="https://hsto.org/webt/_r/ne/yt/_rneytazmfi6nqrka9r5nkdramc.png" alt="screenshot" width="925" />
</p>

### Dependencies

_WIP_

All what you need to start this application - is a [redis server](https://redis.io/), which is running on your host or in docker container.

## Starting

_WIP_

Download compiled application from [releases page][link_releases] _(also you will need to download `./web` directory from this repository)_ or compile from sources and run it locally:

```bash
$ git clone https://github.com/tarampampam/webhook-tester.git ./webhook-tester && cd $_
$ go build -ldflags="-s -w" ./cmd/webhook-tester/
$ ./webhook-tester serve --port 8080 --redis-host 127.0.0.1 --redis-port 6379
```

> For this `redis` server must be installed and started locally on `6379` port. Or you can try to use [cloud version](https://redislabs.com/try-free/) of `redis`.

Or use ready docker image for this. Simple `docker-compose` file below:

```yaml
version: '3.4'

volumes:
  redis-data:

services:
  app:
    image: tarampampam/webhook-tester:latest
    command: serve --port 8080 --redis-host redis
    ports:
      - '8080:8080/tcp' # Open <http://127.0.0.1:8080>

  redis:
    image: redis:6.0.5-alpine
    volumes:
      - redis-data:/data:cached
    ports:
      - 6379
```

> Important notice: do **not** use `latest` application tag _(this is bad practice)_. Use versioned tag (like `1.2.3`) instead.

[![image stats](https://dockeri.co/image/tarampampam/webhook-tester)][link_docker_tags]

All supported image tags [can be found here][link_docker_tags].

### Additional configuration

All supported `serve` command flags can be found by running `docker run --rm -t tarampampam/webhook-tester serve -h`.

### Allowed environment variables

_WIP_

Variable name    | Description
:--------------: | :---------:
`LISTEN_ADDR`    | IP address to listen on
`LISTEN_PORT`    | TCP port number
`PUBLIC_DIR`     | Directory with public assets
`MAX_REQUESTS`   | Maximum stored requests per session
`SESSION_TTL`    | Session lifetime (in seconds)
`REDIS_HOST`     | Redis server hostname or IP address
`REDIS_PORT`     | Redis server TCP port number
`REDIS_PASSWORD` | Redis server password (optional)
`REDIS_DB_NUM`   | Redis database number
`REDIS_MAX_CONN` | Maximum redis connections
`PUSHER_APP_ID`  | Pusher application ID
`PUSHER_KEY`     | Pusher key
`PUSHER_SECRET`  | Pusher secret
`PUSHER_CLUSTER` | Pusher cluster

### Liveness/readiness probes

HTTP get `/live` and `/ready` respectively.

### Testing

For application testing we use built-in golang testing feature and `docker-ce` + `docker-compose` as develop environment. So, just write into your terminal after repository cloning:

```shell
$ make test
```

## Changes log

[![Release date][badge_release_date]][link_releases]
[![Commits since latest release][badge_commits_since_release]][link_commits]

Changes log can be [found here][link_changes_log].

## Releasing

New versions publishing is very simple - just "publish" new release using repo releases page.

> Release version _(and git tag, of course)_ MUST starts with `v` prefix (eg.: `v0.0.1` or `v1.2.3-RC1`)

## Support

[![Issues][badge_issues]][link_issues]
[![Issues][badge_pulls]][link_pulls]

If you will find any package errors, please, [make an issue][link_create_issue] in current repository.

## License

This is open-sourced software licensed under the [MIT License][link_license].

[badge_build]:https://img.shields.io/github/workflow/status/tarampampam/webhook-tester/tests?maxAge=30&logo=github
[badge_release]:https://img.shields.io/github/workflow/status/tarampampam/webhook-tester/release?maxAge=30&label=release&logo=github
[badge_coverage]:https://img.shields.io/codecov/c/github/tarampampam/webhook-tester/master.svg?maxAge=30
[badge_release_version]:https://img.shields.io/github/release/tarampampam/webhook-tester.svg?maxAge=30
[badge_language]:https://img.shields.io/github/go-mod/go-version/tarampampam/webhook-tester?longCache=true
[badge_license]:https://img.shields.io/github/license/tarampampam/webhook-tester.svg?longCache=true
[badge_release_date]:https://img.shields.io/github/release-date/tarampampam/webhook-tester.svg?maxAge=180
[badge_commits_since_release]:https://img.shields.io/github/commits-since/tarampampam/webhook-tester/latest.svg?maxAge=45
[badge_issues]:https://img.shields.io/github/issues/tarampampam/webhook-tester.svg?maxAge=45
[badge_pulls]:https://img.shields.io/github/issues-pr/tarampampam/webhook-tester.svg?maxAge=45

[link_coverage]:https://codecov.io/gh/tarampampam/webhook-tester
[link_build]:https://github.com/tarampampam/webhook-tester/actions
[link_docker_hub]:https://hub.docker.com/r/tarampampam/webhook-tester/
[link_docker_tags]:https://hub.docker.com/r/tarampampam/webhook-tester/tags
[link_license]:https://github.com/tarampampam/webhook-tester/blob/master/LICENSE
[link_releases]:https://github.com/tarampampam/webhook-tester/releases
[link_commits]:https://github.com/tarampampam/webhook-tester/commits
[link_changes_log]:https://github.com/tarampampam/webhook-tester/blob/master/CHANGELOG.md
[link_issues]:https://github.com/tarampampam/webhook-tester/issues
[link_create_issue]:https://github.com/tarampampam/webhook-tester/issues/new/choose
[link_pulls]:https://github.com/tarampampam/webhook-tester/pulls
