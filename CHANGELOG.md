# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

## UNRELEASED

### Added

- E2E test using postman

### Changed

- Docker image based on `scratch` (not `alpine` image)
- Directory `public` renamed to `web`

### Removed

- Binary file packing using `upx`
- HTTP method `CONNECT` for webhook endpoint

### Fixed

- Wrong `content` property value for session creation HTTP handler
- Wrong `Content-Type` header value for webhook handler
- Correct file mime-types for docker image

## v0.2.0

### Changed

- Golang updated from `1.14` up to `1.15`

## v0.1.0

### Added

- Flag `--ignore-header-prefix` for incoming webhook headers ignoring (by prefix)

## v0.0.1

### Added

- Basic features, like UI, `pusher.com`, probes, session and request handlers

[keepachangelog]:https://keepachangelog.com/en/1.0.0/
[semver]:https://semver.org/spec/v2.0.0.html
