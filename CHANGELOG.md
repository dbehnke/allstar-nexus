# Changelog

## [0.12.0](https://github.com/dbehnke/allstar-nexus/compare/v0.11.1...v0.12.0) (2025-11-23)


### Features

* AMI keying improvements and frontend polish ([#93](https://github.com/dbehnke/allstar-nexus/issues/93)) ([ec4e787](https://github.com/dbehnke/allstar-nexus/commit/ec4e787fe639784c148a253cea556126c8dcb992))


### Bug Fixes

* resolve critical stability issues and resource leaks ([#90](https://github.com/dbehnke/allstar-nexus/issues/90)) ([5b317eb](https://github.com/dbehnke/allstar-nexus/commit/5b317eb227948607a8ab70a57ecc06e761ace024))

## [0.11.1](https://github.com/dbehnke/allstar-nexus/compare/v0.11.0...v0.11.1) (2025-11-03)


### Bug Fixes

* GoReleaser nfpm scripts: use file paths instead of inline content ([#75](https://github.com/dbehnke/allstar-nexus/issues/75)) ([fb093d1](https://github.com/dbehnke/allstar-nexus/commit/fb093d1e290996a679cddcead15007405d2c9c89))

## [0.11.0](https://github.com/dbehnke/allstar-nexus/compare/v0.10.1...v0.11.0) (2025-11-03)


### Features

* add systemd service, Makefile install targets, and package generation ([#68](https://github.com/dbehnke/allstar-nexus/issues/68)) ([02ca554](https://github.com/dbehnke/allstar-nexus/commit/02ca5540018b02e6679dd7515a487895f5055cb5))

## [0.10.1](https://github.com/dbehnke/allstar-nexus/compare/v0.10.0...v0.10.1) (2025-11-03)


### Bug Fixes

* use start_unix for transmission log sorting and add cap reset docs ([#64](https://github.com/dbehnke/allstar-nexus/issues/64)) ([ef2a8fb](https://github.com/dbehnke/allstar-nexus/commit/ef2a8fb4d0e3e79350af1e88089584b1d3df3144))

## [0.10.0](https://github.com/dbehnke/allstar-nexus/compare/v0.9.0...v0.10.0) (2025-11-02)


### ⚠ BREAKING CHANGES

* Requires database migration on startup. Existing databases will be automatically upgraded with new columns and indexes.

### Bug Fixes

* timestamp filtering with epoch-based queries ([#58](https://github.com/dbehnke/allstar-nexus/issues/58)) ([2d5aa0b](https://github.com/dbehnke/allstar-nexus/commit/2d5aa0bcbc7f408c9ccaad6aab167f8ef5b923bf))

## [0.9.0](https://github.com/dbehnke/allstar-nexus/compare/v0.8.0...v0.9.0) (2025-11-02)


### Features

* Add Discord webhook notifications for node activity with callsign display and emojis ([#46](https://github.com/dbehnke/allstar-nexus/issues/46)) ([cb0fcea](https://github.com/dbehnke/allstar-nexus/commit/cb0fcea9067ef0656b8a3a2ebb531ae2890bdfec))

## [0.8.0](https://github.com/dbehnke/allstar-nexus/compare/v0.7.0...v0.8.0) (2025-10-29)


### Features

* Add CI/CD automation from dmr-nexus ([#31](https://github.com/dbehnke/allstar-nexus/issues/31)) ([1e62390](https://github.com/dbehnke/allstar-nexus/commit/1e623900de6897f5be9919bc4ade9ec4a4998cf2))

## [0.7.0] - 2025-10-29

### Features

* Initial release with CI/CD automation
* Real-time dashboard for Allstar Link monitoring
* AMI connector with auto-reconnect
* Link tracking and TX batching
* Admin authentication and role-based access
* SQLite persistence with WAL mode
* WebSocket-driven UI updates
* Gamification system with XP and leveling

### Documentation

* Added comprehensive project documentation
* Added quick start guide
* Added configuration examples
