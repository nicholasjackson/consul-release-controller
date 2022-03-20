# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Helm chart Webhook config failure policy now defaults to `Ignore`
- Configuration for the server moved to global `config` package

### Added
- PostDeploymentTests

## [0.0.14 - 2022-03-14
### Fixed
- Ensure a release reconfigures the plugins on update

## [0.0.11 - 2022-03-08
### Changed
- Updated Kubernetes deployment health timeout to 10 minutes from 1 minute.
## [0.0.11 - 2022-03-08
### Added
- Webhooks for Slack and Discord
- Validating admission controller to ensure Kubernetes deployments do not override an active release
- Ability to set custom queries for prometheus

### Fixed
- Fix Helm chart values when TLS not used
- Fix CRDs to make Consul enterprise `namespace` and `partition` optional