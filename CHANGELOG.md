# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.1.0] - 2026-02-03

### Changed
- chore(deps): Update verdacloud-sdk-go from v1.1.2 to v1.2.0
- refactor(Makefile): Auto-load .env file before running integration tests

### Fixed
- fix(resources): Fix time.Time formatting for CreatedAt fields (SDK breaking change)
- fix(instance): Update Delete method signature for new SDK API
- fix(integration): Use correct output attributes for container and serverless_job tests

### Removed
- Remove redundant cleanup-test-resources.sh script (t.Cleanup handles cleanup)

## [v1.0.1] - 2026-01-29

### Fixed
- fix: instance_type docs bug, improve docs, add integration tests

## [v1.0.0] - 2026-01-28

### Added
- feat: Configure GoReleaser for Terraform Registry
- feat: Add Terraform Registry publishing support

## [v0.1.0] - 2025-12-10

### Added
- Initial beta release with core resources:
  - `verda_instance` - GPU instance management
  - `verda_volume` - Storage volume management
  - `verda_ssh_key` - SSH key management
  - `verda_startup_script` - Startup script management
  - `verda_container` - Serverless container deployment
  - `verda_serverless_job` - Serverless job deployment
  - `verda_container_registry_credentials` - Registry credentials management
