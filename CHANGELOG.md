# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
