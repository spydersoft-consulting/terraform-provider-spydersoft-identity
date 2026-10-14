# Changelog

All notable changes to this provider are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Resource `spydersoftidentity_role` (name and claims, import by name) and data source
  `spydersoftidentity_role`. Role claim types must not contain `/`: the admin API
  cannot delete such a claim, so the provider rejects them at plan time.
- Resources: `spydersoftidentity_scope`, `spydersoftidentity_api_resource`,
  `spydersoftidentity_identity_resource`, `spydersoftidentity_client`,
  `spydersoftidentity_client_secret`, `spydersoftidentity_api_resource_secret`.
- Data sources: `spydersoftidentity_scope`, `spydersoftidentity_api_resource`,
  `spydersoftidentity_identity_resource`, `spydersoftidentity_client`.
- Import by natural key (name or `client_id`) for scopes, API resources,
  identity resources and clients.
