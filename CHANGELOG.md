# Changelog

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
