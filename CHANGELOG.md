# Changelog

## [0.15.4](https://github.com/dbehnke/allstar-nexus/compare/v0.15.3...v0.15.4) (2026-04-18)


### Bug Fixes

* **release:** remove unsupported merge plugin ([#164](https://github.com/dbehnke/allstar-nexus/issues/164)) ([2d178a5](https://github.com/dbehnke/allstar-nexus/commit/2d178a5195a296e187c2b8905741e4a1cd916236))
* **release:** use release-please-oss/release-please-action v5 ([#166](https://github.com/dbehnke/allstar-nexus/issues/166)) ([f021696](https://github.com/dbehnke/allstar-nexus/commit/f021696f8a9b58c1f88875c22b91026bffaef37a))
* **state:** filter AMI events by Node: header to prevent shared-server broadcast crosstalk ([#161](https://github.com/dbehnke/allstar-nexus/issues/161)) ([aa4f0ad](https://github.com/dbehnke/allstar-nexus/commit/aa4f0ad9c609d0518acabef3ac9daf4196b6c033))

## [0.15.3](https://github.com/dbehnke/allstar-nexus/compare/v0.15.2...v0.15.3) (2026-04-18)


### Bug Fixes

* **keying:** clear adjacentNodes on ProcessALinks and filter seed by LocalNode ([c5aa10f](https://github.com/dbehnke/allstar-nexus/commit/c5aa10f20ea2c6407016a91b3397c49f8849d01e))
* **keying:** clear adjacentNodes on ProcessALinks and filter seed by LocalNode ([4c9aaa4](https://github.com/dbehnke/allstar-nexus/commit/4c9aaa49657139755ddc1dccb3344d320a01d589))
* **release:** remove unsupported merge plugin ([#164](https://github.com/dbehnke/allstar-nexus/issues/164)) ([2d178a5](https://github.com/dbehnke/allstar-nexus/commit/2d178a5195a296e187c2b8905741e4a1cd916236))
* **state:** distinguish zero from missing in per-source adjacent counter fallback ([3f2d5d2](https://github.com/dbehnke/allstar-nexus/commit/3f2d5d2693b43d8e85b8d1dadfd513b505455006))
* **state:** filter AMI events by Node: header to prevent shared-server broadcast crosstalk ([#161](https://github.com/dbehnke/allstar-nexus/issues/161)) ([aa4f0ad](https://github.com/dbehnke/allstar-nexus/commit/aa4f0ad9c609d0518acabef3ac9daf4196b6c033))

## [0.15.2](https://github.com/dbehnke/allstar-nexus/compare/v0.15.1...v0.15.2) (2026-04-14)


### Bug Fixes

* **keying:** clear adjacentNodes on ProcessALinks and filter seed by LocalNode ([c5aa10f](https://github.com/dbehnke/allstar-nexus/commit/c5aa10f20ea2c6407016a91b3397c49f8849d01e))
* **keying:** clear adjacentNodes on ProcessALinks and filter seed by LocalNode ([4c9aaa4](https://github.com/dbehnke/allstar-nexus/commit/4c9aaa49657139755ddc1dccb3344d320a01d589))
* **state:** distinguish zero from missing in per-source adjacent counter fallback ([3f2d5d2](https://github.com/dbehnke/allstar-nexus/commit/3f2d5d2693b43d8e85b8d1dadfd513b505455006))
* **state:** distinguish zero from missing in per-source adjacent counter fallback ([3545e4d](https://github.com/dbehnke/allstar-nexus/commit/3545e4d892fd4e7653d58f7891a7bf0ec6fb3b15))

## [0.15.1](https://github.com/dbehnke/allstar-nexus/compare/v0.15.0...v0.15.1) (2026-04-14)


### Bug Fixes

* wire TriggerPoll and tag links to correct AMI source node ([#155](https://github.com/dbehnke/allstar-nexus/issues/155)) ([607caa9](https://github.com/dbehnke/allstar-nexus/commit/607caa946cc943d45daa1bff8459cf1779e94c06))

## [0.15.0](https://github.com/dbehnke/allstar-nexus/compare/v0.14.2...v0.15.0) (2026-04-13)


### Features

* **ami:** add SourceNodeID to Message struct ([c831b05](https://github.com/dbehnke/allstar-nexus/commit/c831b059d948f600354d10bc0d2e0668b4dd4262))
* **ami:** tag messages with sourceNodeID in connector ([b5f1fc2](https://github.com/dbehnke/allstar-nexus/commit/b5f1fc2f54178c31400c047093d996e78a491563))
* **config:** add Visible, AMIHost, AMIPort to NodeConfig ([103a56b](https://github.com/dbehnke/allstar-nexus/commit/103a56b7799e3ab0c164337ca5cd97e015652742))
* **gamification:** add ExcludedCallsigns to config structs ([12a6ca4](https://github.com/dbehnke/allstar-nexus/commit/12a6ca4179cd3517347c217712bbeb8a5203f5e9))
* **gamification:** add ScoringSourceNodeID to config structs ([4b73590](https://github.com/dbehnke/allstar-nexus/commit/4b735909c24e02f3ea6c314cf8db4f6bcdec1e74))
* **gamification:** exclude callsigns from XP scoring ([ec54d33](https://github.com/dbehnke/allstar-nexus/commit/ec54d33e3332dd798181d7de9a7f3fdc6ae4f7c4))
* multi-node gamification and UI — all features ([7766ced](https://github.com/dbehnke/allstar-nexus/commit/7766ced5a95c3ef5acd193b0cb9b64dce2d3e760))
* **multi-node:** create per-node AMI connectors with sourceNodeID tagging ([e9a6815](https://github.com/dbehnke/allstar-nexus/commit/e9a68156bb66a1641fdc84f5bf3e9640a19e7d56))


### Bug Fixes

* **core:** remove early return in ALINKS handler so LINKS handler can still populate LinksDetailed ([68fe322](https://github.com/dbehnke/allstar-nexus/commit/68fe322d90cae3f1874aeddc69415d9da82bcc89))
* **core:** route ALINKS events only to matching keyingTracker ([b2e27b8](https://github.com/dbehnke/allstar-nexus/commit/b2e27b8a24411e7165c0bd0934ab9e1567bfeec2))

## [0.14.2](https://github.com/dbehnke/allstar-nexus/compare/v0.14.1...v0.14.2) (2026-04-11)


### Bug Fixes

* golangci-lint version prefix and npm audit issues ([453af27](https://github.com/dbehnke/allstar-nexus/commit/453af27b4579814f619deaea4f864cba3257e904))
* remove invalid linters-settings from .golangci.yml (v2 schema) ([b740c19](https://github.com/dbehnke/allstar-nexus/commit/b740c194d47a25aaad8cfb4f469d29778a6a7c1f))
* restore v prefix on golangci-lint version for v7 action ([5b09ebb](https://github.com/dbehnke/allstar-nexus/commit/5b09ebb0ec51ce83f7a365dcea9feeda61bace2b))
* update golangci-lint config to v2 format ([0822d75](https://github.com/dbehnke/allstar-nexus/commit/0822d75c127266727091d4f106d557f8cd10a44a))
* upgrade to golangci-lint-action@v7 (v6 doesn't support v2 linter) ([55eb6dd](https://github.com/dbehnke/allstar-nexus/commit/55eb6ddec0af0875762545ac60bbd4c679396fd3))
* use golangci-lint-action@v7 for golangci-lint v2 ([118ff63](https://github.com/dbehnke/allstar-nexus/commit/118ff63e0c63524f2217b2bb7626098a42d31408))

## [0.14.1](https://github.com/dbehnke/allstar-nexus/compare/v0.14.0...v0.14.1) (2026-01-23)


### Bug Fixes

* config.yaml tab indentation causing AMI settings to be ignored ([#124](https://github.com/dbehnke/allstar-nexus/issues/124)) ([b2c4ce0](https://github.com/dbehnke/allstar-nexus/commit/b2c4ce0a9420da23815f185e93dc98d5fb88c4a4))

## [0.14.0](https://github.com/dbehnke/allstar-nexus/compare/v0.13.0...v0.14.0) (2026-01-11)


### Features

* **urfd:** implement NNG control client for USRP registration ([#117](https://github.com/dbehnke/allstar-nexus/issues/117)) ([9465867](https://github.com/dbehnke/allstar-nexus/commit/9465867739441cdc4a28a5647d1c7bacfbd00706))

## [0.13.0](https://github.com/dbehnke/allstar-nexus/compare/v0.12.0...v0.13.0) (2025-12-21)


### Features

* migrate build system to Taskfile.dev and establish development rules ([#108](https://github.com/dbehnke/allstar-nexus/issues/108)) ([66c7300](https://github.com/dbehnke/allstar-nexus/commit/66c7300723aae0b39eee7fe2bcdab935854fb0ae))


### Bug Fixes

* prevent callsign chopping for DVSwitch/text nodes ([#112](https://github.com/dbehnke/allstar-nexus/issues/112)) ([37e1252](https://github.com/dbehnke/allstar-nexus/commit/37e1252cf9bc5ecf9d1010fc2d82643a759cfa0b))

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
