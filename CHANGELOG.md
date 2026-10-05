# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.12](https://github.com/SpechtLabs/golint-sl/compare/v0.1.11...v0.1.12) (2026-10-05)


### Features

* **pre-commit:** lint every Go module of the repository ([#126](https://github.com/SpechtLabs/golint-sl/issues/126)) ([7c37bcc](https://github.com/SpechtLabs/golint-sl/commit/7c37bcc57f0a3bb6c94e25dd9652bf97615c10a0))
* publish golint-sl via homebrew tap ([#65](https://github.com/SpechtLabs/golint-sl/issues/65)) ([b5c6d33](https://github.com/SpechtLabs/golint-sl/commit/b5c6d33a363d9ce2629a6aa80dc36bb08d633a32))


### Bug Fixes

* **clockinterface:** exempt by package name and whole words, and resolve time calls with types ([#138](https://github.com/SpechtLabs/golint-sl/issues/138)) ([814e084](https://github.com/SpechtLabs/golint-sl/commit/814e084c5de7108bcf79d4317760e4c4edef6672))
* **closurecomplexity:** count statements, captures and nesting correctly ([#143](https://github.com/SpechtLabs/golint-sl/issues/143)) ([f0b51a6](https://github.com/SpechtLabs/golint-sl/commit/f0b51a6beb894c3a4b4485f056900534290c28ca))
* **contextfirst:** report the real parameter position and match only context.Context ([#137](https://github.com/SpechtLabs/golint-sl/issues/137)) ([1026e02](https://github.com/SpechtLabs/golint-sl/commit/1026e023ef5c80aac0c3b8ca55525f91220d0b7a))
* **contextlogger:** report each global logger call once and match loggers by package ([#147](https://github.com/SpechtLabs/golint-sl/issues/147)) ([c7625de](https://github.com/SpechtLabs/golint-sl/commit/c7625defae9f873505ecf9ebe80467ce5bed9b73))
* **contextpropagation:** report a call once and only suggest XContext methods that exist ([#158](https://github.com/SpechtLabs/golint-sl/issues/158)) ([fcb09e2](https://github.com/SpechtLabs/golint-sl/commit/fcb09e2add32af14d7d2004b2313f41e848136d5))
* **dataflow:** follow variadic arguments to log calls and match only real logging packages ([#159](https://github.com/SpechtLabs/golint-sl/issues/159)) ([a930514](https://github.com/SpechtLabs/golint-sl/commit/a9305145a058d503ef10d8cf0e35c4c0279db8f8))
* **deps:** require Go 1.27 and update golang.org/x/tools to v0.51.0 ([#125](https://github.com/SpechtLabs/golint-sl/issues/125)) ([d2a2eac](https://github.com/SpechtLabs/golint-sl/commit/d2a2eac133656ef42a0643a368569de0c3fb3e36))
* **deps:** update module golang.org/x/tools to v0.45.0 ([#50](https://github.com/SpechtLabs/golint-sl/issues/50)) ([f9201e3](https://github.com/SpechtLabs/golint-sl/commit/f9201e32164af8b907c31434d3ab5ee081293656))
* **deps:** update module golang.org/x/tools to v0.48.0 ([#62](https://github.com/SpechtLabs/golint-sl/issues/62)) ([da2fe88](https://github.com/SpechtLabs/golint-sl/commit/da2fe883eece5442573202969124e223e8aaedd5))
* **deps:** update module golang.org/x/tools to v0.49.0 ([#90](https://github.com/SpechtLabs/golint-sl/issues/90)) ([e207e5e](https://github.com/SpechtLabs/golint-sl/commit/e207e5e63f9afc1edb50cac4a07a79e8114474d6))
* **emptyinterface:** name the actual type in messages and drop the type assertion promise ([#135](https://github.com/SpechtLabs/golint-sl/issues/135)) ([93cc785](https://github.com/SpechtLabs/golint-sl/commit/93cc785cd9964f5b312f0d9f257a602b1d4ffa66))
* **errorwrap:** check each return on its own and inspect naked returns ([#145](https://github.com/SpechtLabs/golint-sl/issues/145)) ([5ef366a](https://github.com/SpechtLabs/golint-sl/commit/5ef366ace431e2410c784653bfb791c5e9f8cefc))
* **exporteddoc:** accept block comment docs and check standalone variable docs ([#148](https://github.com/SpechtLabs/golint-sl/issues/148)) ([2c8c41c](https://github.com/SpechtLabs/golint-sl/commit/2c8c41c9b976160415056d5ce3cadfd086a17cbf))
* **functionsize:** measure nesting depth so the early-return advice can appear ([#151](https://github.com/SpechtLabs/golint-sl/issues/151)) ([5bd3e56](https://github.com/SpechtLabs/golint-sl/commit/5bd3e56a4f47dc488508cfe038366fe3c7a4e981))
* **goroutineleak:** look up the enclosing function's context per go statement and treat for true as infinite ([#153](https://github.com/SpechtLabs/golint-sl/issues/153)) ([e38cdc1](https://github.com/SpechtLabs/golint-sl/commit/e38cdc1ce096a1b951363a403631c9c2f3136573))
* **hardcodedcreds:** match credential words in names and detect base64 and connection string credentials ([#160](https://github.com/SpechtLabs/golint-sl/issues/160)) ([6395eb2](https://github.com/SpechtLabs/golint-sl/commit/6395eb215540d1cb301fa70070e641b72f8a8bf2))
* **httpclient:** resolve net/http through the type checker and read unkeyed client literals ([#163](https://github.com/SpechtLabs/golint-sl/issues/163)) ([7ad313d](https://github.com/SpechtLabs/golint-sl/commit/7ad313dcc51e5a5d64da373d79049c8434c9686e))
* **humaneerror:** scope the fmt.Errorf callback exemption to its own function ([#140](https://github.com/SpechtLabs/golint-sl/issues/140)) ([11e3489](https://github.com/SpechtLabs/golint-sl/commit/11e348993de46824e00a6fca03a0a9d259e21a4e))
* **interfaceconsistency:** report each dependency once and stop asking constructors to return interfaces ([#144](https://github.com/SpechtLabs/golint-sl/issues/144)) ([4042bdc](https://github.com/SpechtLabs/golint-sl/commit/4042bdc443163c2d9bafd858a2097d3150f60bdc))
* **lifecycle:** only require ctx.Done() for long-running loops ([#150](https://github.com/SpechtLabs/golint-sl/issues/150)) ([3d034ec](https://github.com/SpechtLabs/golint-sl/commit/3d034ecae84ea9d5ffd8ce00b6428c7a06bf6e55))
* **mockverify:** only check mock types declared in mock files ([#139](https://github.com/SpechtLabs/golint-sl/issues/139)) ([d89f55d](https://github.com/SpechtLabs/golint-sl/commit/d89f55d10324a5a930d1db5d98dc5f3b4bf009f9))
* **nestingdepth:** report each if-else chain once, at its head ([#142](https://github.com/SpechtLabs/golint-sl/issues/142)) ([800f0d3](https://github.com/SpechtLabs/golint-sl/commit/800f0d376b6e209117ba6254417c040593fe1a51))
* **nilcheck:** accept compound checks and only count checks that guard the use ([#154](https://github.com/SpechtLabs/golint-sl/issues/154)) ([15ba36b](https://github.com/SpechtLabs/golint-sl/commit/15ba36b390a32eb143e6ed99ce2c46e3a73272ab))
* **nopanic:** match fatal loggers by type and stop treating code after init as init ([#146](https://github.com/SpechtLabs/golint-sl/issues/146)) ([ca889b4](https://github.com/SpechtLabs/golint-sl/commit/ca889b4ad543b31e0df8b4326d7ae441f8b264fa))
* **optionspattern:** only treat functional option types as options ([#149](https://github.com/SpechtLabs/golint-sl/issues/149)) ([f529d59](https://github.com/SpechtLabs/golint-sl/commit/f529d59e8ec77ee3f5bb2535f4e8e700d1c2c8e5))
* **pkgnaming:** only check exported package-level types for stutter ([#134](https://github.com/SpechtLabs/golint-sl/issues/134)) ([bf51bd1](https://github.com/SpechtLabs/golint-sl/commit/bf51bd1a78e9f9ffb227a0f4796bb4a68958b867))
* **reconciler:** accept client.IgnoreNotFound and match HTTP, database and mutex calls by type ([#161](https://github.com/SpechtLabs/golint-sl/issues/161)) ([e0933f6](https://github.com/SpechtLabs/golint-sl/commit/e0933f6adf36374d92d0e0de549b0ab977ee0940))
* reject golint-sl settings that can't be decoded or name no analyzer ([#166](https://github.com/SpechtLabs/golint-sl/issues/166)) ([0d54234](https://github.com/SpechtLabs/golint-sl/commit/0d542347b7716b3609a84aa6891d4e7c608fa83f))
* **resourceclose:** accept returned and return-closed resources, catch typed dials and shadowing ([#141](https://github.com/SpechtLabs/golint-sl/issues/141)) ([9c82aaf](https://github.com/SpechtLabs/golint-sl/commit/9c82aafe61c3bc6941d053b71182f98cc1cc14f0))
* **returninterface:** skip type parameters and match exempt interfaces by package path ([#155](https://github.com/SpechtLabs/golint-sl/issues/155)) ([c659861](https://github.com/SpechtLabs/golint-sl/commit/c659861ea7fc72d114f47ec66c93de761fb9f331))
* **sentinelerrors:** stop reporting package-level sentinel errors ([#157](https://github.com/SpechtLabs/golint-sl/issues/157)) ([4e8723b](https://github.com/SpechtLabs/golint-sl/commit/4e8723be8cef74dd41ef368b4e64bba39fe104ec))
* **sideeffects:** don't crash GetAllFunctions on a type alias ([#133](https://github.com/SpechtLabs/golint-sl/issues/133)) ([baa7c57](https://github.com/SpechtLabs/golint-sl/commit/baa7c574c2266cb24bf6c68ae2edfafbc472552e))
* **sideeffects:** follow variadic logger arguments and match the configured patterns ([#165](https://github.com/SpechtLabs/golint-sl/issues/165)) ([0d60112](https://github.com/SpechtLabs/golint-sl/commit/0d60112196b9f2857500abb274224e398bf14ef0))
* **sideeffects:** stop TrackDataFlow recursing forever on loops ([#132](https://github.com/SpechtLabs/golint-sl/issues/132)) ([e687de9](https://github.com/SpechtLabs/golint-sl/commit/e687de95ff430a69df2e687eee7db82bc1d227b2))
* **statusupdate:** count status writes through a variable holding the status writer ([#164](https://github.com/SpechtLabs/golint-sl/issues/164)) ([6424c67](https://github.com/SpechtLabs/golint-sl/commit/6424c67314f5ba328e2d0ab7e564567918627079))
* **statusupdate:** only count writes through the status subresource as status updates ([#162](https://github.com/SpechtLabs/golint-sl/issues/162)) ([489c8a0](https://github.com/SpechtLabs/golint-sl/commit/489c8a042ea35a528971d08b0ee319cec7e1d6be))
* **syncaccess:** don't crash on a method without a body ([#131](https://github.com/SpechtLabs/golint-sl/issues/131)) ([cbe074b](https://github.com/SpechtLabs/golint-sl/commit/cbe074b106da4346cc0ee905ba5859a54e7b6152))
* **syncaccess:** match captures by identity and honor locks and per-iteration loop variables ([#152](https://github.com/SpechtLabs/golint-sl/issues/152)) ([654578b](https://github.com/SpechtLabs/golint-sl/commit/654578bb4585e651dc1298dbf05fff304dcfc77e))
* **todotracker:** only flag TODO and FIXME words that start a comment line ([#136](https://github.com/SpechtLabs/golint-sl/issues/136)) ([186a49f](https://github.com/SpechtLabs/golint-sl/commit/186a49f355be69b8eeef4d61dd39c475d0d8538b))
* **wideevents:** recognize log calls by logger type and count chained fields, loops and context fields correctly ([#156](https://github.com/SpechtLabs/golint-sl/issues/156)) ([aa6fe5b](https://github.com/SpechtLabs/golint-sl/commit/aa6fe5bade3821a6c853d273323d20bf59653390))

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
