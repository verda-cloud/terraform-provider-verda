# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- feat(managed-kubernetes): Experimental managed Kubernetes support with three new resources. `verda_managed_kubernetes_cluster`: clusters are referenced by the composite ID `<location>/<cluster-name>`, only `kubernetes_version` updates in place (rolling control-plane upgrade), create/update wait for phase `running` and delete waits until the cluster is gone (configurable `timeouts = { ... }`, 30m defaults). `verda_managed_kubernetes_node_pool`: pools reference their cluster via `cluster_id`, only `replicas` updates in place (`0` scales down), and the computed `errors` attribute surfaces server-side failure reasons (`insufficient_capacity`, `insufficient_funds`, ...). `verda_managed_kubernetes_kubeconfig`: issues a client certificate — any change re-issues it by replacement — and exposes the rendered YAML as the sensitive `content` attribute; deletion does not revoke downloaded kubeconfigs (valid until `not_after`), and rotation is manual (taint and re-apply). All three resources support import via composite IDs.

### Changed

- chore(deps): Pin verdacloud-sdk-go to the unreleased managed Kubernetes commit 7c26c3b (pseudo-version `v1.7.2-0.20261104131330-7c26c3b0f835`) until a tagged SDK release ships it; the SDK's `go` directive raises the minimum toolchain to Go 1.26.10.

## [v1.1.4] - 2026-09-24

### Fixed

- fix(instances): `terraform destroy` deletes the instance's OS volume again. `Delete` sent `volume_ids: []`, which since verdacloud-sdk-go v1.4.0 means "delete no volume" and left every OS volume detached and billed (#17). New `os_volume.on_destroy`: `delete_permanently` (default), `move_to_trash`, `keep_detached`; changing it updates state in place without touching the API. Adding or removing the whole `os_volume` object still replaces the instance.
- docs(instances): the `os_volume` and `volumes` examples and integration testdata use attribute syntax (`os_volume = { ... }`, `volumes = [{ ... }]`); block syntax fails validation.
- fix(instances): `verda_instance` create waits until the instance is `running` with an address, so `ip` is known to provisioners and dependents on the first apply; before, the resource returned as soon as the API accepted the order (`provisioning`, no address). New `timeouts { create = "20m" }`; changing it later updates state in place without touching the API, like `os_volume.on_destroy`. A terminal status (`error`, `discontinued`, `notfound`, `deleting`) fails the apply, as does running out of `timeouts.create`; the instance is kept in state as tainted, so the next apply replaces it (`terraform untaint` keeps it). `Read` keeps a known `ip` when the API momentarily reports none for a running instance.

## [v1.1.2] - 2026-02-26

### Added

- feat(volumes): Add new resource for managing shared filesystem volumes (verda_volume_attachment)

### Fixed

- fix(instances): Support multiple SSH keys without crashing

## [v1.1.1] - 2026-02-05

### Added

- feat(provider): Add User-Agent header to API requests (`verda-terraform-provider/{provider_version}(terraform/{terraform_version})`)

### Changed

- chore(deps): Update verdacloud-sdk-go from v1.2.0 to v1.2.1

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
