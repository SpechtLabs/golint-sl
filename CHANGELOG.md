# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.6](https://github.com/SpechtLabs/golint-sl/compare/v0.1.5...v0.1.6) (2026-01-16)


### Features

* add nolint directive support and fix humaneerror detection ([1e1641a](https://github.com/SpechtLabs/golint-sl/commit/1e1641a3a0c0ffe4d6e83f34ee7ac3b2923ab2e4))
* Initial commit ([9fafe39](https://github.com/SpechtLabs/golint-sl/commit/9fafe3911ea8629fd7ed7914493c1b28c8cc86c0))


### Bug Fixes

* address golangci-lint issues ([1b0e689](https://github.com/SpechtLabs/golint-sl/commit/1b0e6892c1b697d32098c9c2141140e0b3eecbe9))
* Dockerfile ([d360c7d](https://github.com/SpechtLabs/golint-sl/commit/d360c7d8917aa872a9068daf12d6579a1ca4fca4))
* downgrade Go version to 1.24 for golangci-lint compatibility ([ff2477e](https://github.com/SpechtLabs/golint-sl/commit/ff2477ecaecbdb5bba7abbb0ea64017a322df90a))
* pass reporter to checkBranchOnlyVars function ([50c7d2b](https://github.com/SpechtLabs/golint-sl/commit/50c7d2b8a9b602252e26395b363ce4f3416bf3f0))
* remove unused pass parameter from checkBranchOnlyVars ([1d21d63](https://github.com/SpechtLabs/golint-sl/commit/1d21d6301208b18849f2f13597fecf2c3bcc52f8))
* split archives by format to fix homebrew tap release ([d825e53](https://github.com/SpechtLabs/golint-sl/commit/d825e53a2e9a7675a5b6ddb6f410ca6d1433652c))

## [0.1.4](https://github.com/SpechtLabs/golint-sl/compare/v0.1.3...v0.1.4) (2026-01-16)


### Features

* add nolint directive support and fix humaneerror detection ([1e1641a](https://github.com/SpechtLabs/golint-sl/commit/1e1641a3a0c0ffe4d6e83f34ee7ac3b2923ab2e4))


### Bug Fixes

* address golangci-lint issues ([1b0e689](https://github.com/SpechtLabs/golint-sl/commit/1b0e6892c1b697d32098c9c2141140e0b3eecbe9))
* downgrade Go version to 1.24 for golangci-lint compatibility ([ff2477e](https://github.com/SpechtLabs/golint-sl/commit/ff2477ecaecbdb5bba7abbb0ea64017a322df90a))
* pass reporter to checkBranchOnlyVars function ([50c7d2b](https://github.com/SpechtLabs/golint-sl/commit/50c7d2b8a9b602252e26395b363ce4f3416bf3f0))
* remove unused pass parameter from checkBranchOnlyVars ([1d21d63](https://github.com/SpechtLabs/golint-sl/commit/1d21d6301208b18849f2f13597fecf2c3bcc52f8))

## [0.1.3](https://github.com/SpechtLabs/golint-sl/compare/v0.1.2...v0.1.3) (2026-01-16)


### Bug Fixes

* Dockerfile ([d360c7d](https://github.com/SpechtLabs/golint-sl/commit/d360c7d8917aa872a9068daf12d6579a1ca4fca4))

## [0.1.2](https://github.com/SpechtLabs/golint-sl/compare/v0.1.1...v0.1.2) (2026-01-16)


### Bug Fixes

* split archives by format to fix homebrew tap release ([d825e53](https://github.com/SpechtLabs/golint-sl/commit/d825e53a2e9a7675a5b6ddb6f410ca6d1433652c))

## [0.1.1](https://github.com/SpechtLabs/golint-sl/compare/v0.1.0...v0.1.1) (2026-01-16)


### Features

* Initial commit ([9fafe39](https://github.com/SpechtLabs/golint-sl/commit/9fafe3911ea8629fd7ed7914493c1b28c8cc86c0))

## [Unreleased]

### Added

- Initial release with 32 analyzers for Go best practices
- Error handling analyzers: `humaneerror`, `errorwrap`, `sentinelerrors`
- Observability analyzers: `wideevents`, `contextlogger`, `contextpropagation`
- Kubernetes analyzers: `reconciler`, `statusupdate`, `sideeffects`
- Testability analyzers: `clockinterface`, `interfaceconsistency`, `mockverify`, `optionspattern`
- Resource analyzers: `resourceclose`, `httpclient`
- Safety analyzers: `goroutineleak`, `nilcheck`, `nopanic`, `nestingdepth`, `syncaccess`
- Clean code analyzers: `varscope`, `closurecomplexity`, `emptyinterface`, `returninterface`
- Architecture analyzers: `contextfirst`, `pkgnaming`, `functionsize`, `exporteddoc`, `todotracker`, `hardcodedcreds`, `lifecycle`, `dataflow`
- golangci-lint plugin support
- Homebrew formula
- Docker image support
- GitHub Actions CI/CD with release-please
