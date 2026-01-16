# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
