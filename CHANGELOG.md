# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.12](https://github.com/SpechtLabs/golint-sl/compare/v0.1.11...v0.1.12) (2026-09-06)


### Bug Fixes

* **deps:** update dependency mermaid to ^11.16.0 ([#70](https://github.com/SpechtLabs/golint-sl/issues/70)) ([3e960da](https://github.com/SpechtLabs/golint-sl/commit/3e960da06a2f013f4e394289c73b52eee1e15d09))
* **deps:** update dependency mermaid to ^11.16.1 ([#85](https://github.com/SpechtLabs/golint-sl/issues/85)) ([885889b](https://github.com/SpechtLabs/golint-sl/commit/885889bb45ccebc6842bbc5007e558fbfcb31ba4))
* **deps:** update dependency mermaid to ^11.17.0 ([#94](https://github.com/SpechtLabs/golint-sl/issues/94)) ([cf3c742](https://github.com/SpechtLabs/golint-sl/commit/cf3c742db16eb8ddec0b5681188a738a64c852ac))
* **deps:** update dependency mermaid to ^11.17.2 ([#99](https://github.com/SpechtLabs/golint-sl/issues/99)) ([ba38003](https://github.com/SpechtLabs/golint-sl/commit/ba380038b0179a82feb6ea8dbe04cb72566d550f))
* **deps:** update module golang.org/x/tools to v0.45.0 ([#50](https://github.com/SpechtLabs/golint-sl/issues/50)) ([f9201e3](https://github.com/SpechtLabs/golint-sl/commit/f9201e32164af8b907c31434d3ab5ee081293656))
* **deps:** update module golang.org/x/tools to v0.48.0 ([#62](https://github.com/SpechtLabs/golint-sl/issues/62)) ([da2fe88](https://github.com/SpechtLabs/golint-sl/commit/da2fe883eece5442573202969124e223e8aaedd5))
* **deps:** update module golang.org/x/tools to v0.49.0 ([#90](https://github.com/SpechtLabs/golint-sl/issues/90)) ([e207e5e](https://github.com/SpechtLabs/golint-sl/commit/e207e5e63f9afc1edb50cac4a07a79e8114474d6))

## [0.1.11](https://github.com/SpechtLabs/golint-sl/compare/v0.1.10...v0.1.11) (2026-05-14)


### Features

* **humaneerror:** support humane.Newf and humane.Wrapf ([#37](https://github.com/SpechtLabs/golint-sl/issues/37)) ([755567a](https://github.com/SpechtLabs/golint-sl/commit/755567ac7f89b9d7d517a5d23501896ca55b9b19))


### Bug Fixes

* **deps:** update module github.com/golangci/plugin-module-register to v0.1.2 ([#20](https://github.com/SpechtLabs/golint-sl/issues/20)) ([f7e9c1e](https://github.com/SpechtLabs/golint-sl/commit/f7e9c1e11db55e1ef53fb6a92eb2f25648e96a64))
* **deps:** update module golang.org/x/tools to v0.43.0 ([#34](https://github.com/SpechtLabs/golint-sl/issues/34)) ([748894a](https://github.com/SpechtLabs/golint-sl/commit/748894a3edb2c225b2e5051f1500d578b0dfd793))

## [0.1.10](https://github.com/SpechtLabs/golint-sl/compare/v0.1.9...v0.1.10) (2026-03-23)


### Bug Fixes

* reduce false positives in interfaceconsistency and optionspattern analyzers ([#15](https://github.com/SpechtLabs/golint-sl/issues/15)) ([10d0ca8](https://github.com/SpechtLabs/golint-sl/commit/10d0ca81cd180a8928c19ddf0ca058378a9cf35e))

## [0.1.9](https://github.com/SpechtLabs/golint-sl/compare/v0.1.8...v0.1.9) (2026-01-26)


### Bug Fixes

* remove homebrew tap from goreleaser ([5ed3ca0](https://github.com/SpechtLabs/golint-sl/commit/5ed3ca0f438dd959184a93ebbbd24e7b047c31cd))

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
