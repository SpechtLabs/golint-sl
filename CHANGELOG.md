# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.8](https://github.com/SpechtLabs/golint-sl/compare/v0.1.7...v0.1.8) (2026-01-24)


### Features

* **plugin:** add golangci-lint v2 module plugin support ([3923c93](https://github.com/SpechtLabs/golint-sl/commit/3923c93a9c4c0500415256f645f2fb1164096ae3))
* **plugin:** add golangci-lint v2 module plugin support ([ec7672c](https://github.com/SpechtLabs/golint-sl/commit/ec7672c74ceb0166a6b97927357ee1f9923dbd0e))

## [0.1.7](https://github.com/SpechtLabs/golint-sl/compare/v0.1.6...v0.1.7) (2026-01-17)


### Features

* **errorwrap:** skip functions returning humane.Error ([35f8448](https://github.com/SpechtLabs/golint-sl/commit/35f8448478b4d409942b4191d613c0d1b0d598c3))
* **wideevents:** add otelzap support and context-aware method detection ([c5dc27a](https://github.com/SpechtLabs/golint-sl/commit/c5dc27aaa2e1eb3b22becd2fbb90e10b6f36cf59))
* **wideevents:** enforce span attributes when context is available ([2d7ab0c](https://github.com/SpechtLabs/golint-sl/commit/2d7ab0cf09e339ae195353d1933ad0794e540c6a))


### Bug Fixes

* **ci:** install golangci-lint v2 in release workflow ([c41bbcc](https://github.com/SpechtLabs/golint-sl/commit/c41bbcc029d7a912ed8c366aede9391634809637))
* **ci:** install golangci-lint v2 manually ([f5c2271](https://github.com/SpechtLabs/golint-sl/commit/f5c22719d13bb5d3b9c33a5f50c3c0d08e0c255e))
* reduce more false positives in analyzers ([dee1e4f](https://github.com/SpechtLabs/golint-sl/commit/dee1e4f27365a1a4006bb660a067296cfb7e421b))
* **resourceclose:** reduce false positives and improve detection ([896bcea](https://github.com/SpechtLabs/golint-sl/commit/896bceabe6379d0795358fd7849a7cdff7c16dc2))
* **wideevents:** more false positive fixes ([e260b97](https://github.com/SpechtLabs/golint-sl/commit/e260b97c2719cbf3cfa9e5c698eba8b52d32ae86))
* **wideevents:** reduce false positives ([07e88fc](https://github.com/SpechtLabs/golint-sl/commit/07e88fc77676be60c1c97ffe771055929cb124a8))

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

- Initial release with 31 analyzers for Go best practices
- Error handling analyzers: `humaneerror`, `errorwrap`, `sentinelerrors`
- Observability analyzers: `wideevents`, `contextlogger`, `contextpropagation`
- Kubernetes analyzers: `reconciler`, `statusupdate`, `sideeffects`
- Testability analyzers: `clockinterface`, `interfaceconsistency`, `mockverify`, `optionspattern`
- Resource analyzers: `resourceclose`, `httpclient`
- Safety analyzers: `goroutineleak`, `nilcheck`, `nopanic`, `nestingdepth`, `syncaccess`
- Clean code analyzers: `closurecomplexity`, `emptyinterface`, `returninterface`
- Architecture analyzers: `contextfirst`, `pkgnaming`, `functionsize`, `exporteddoc`, `todotracker`, `hardcodedcreds`, `lifecycle`, `dataflow`
- golangci-lint plugin support
- Homebrew formula
- Docker image support
- GitHub Actions CI/CD with release-please
