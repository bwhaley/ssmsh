# Changelog

## [2.0.0] - 2026-09-24

### Highlights

- Upgrade the project to Go 1.27.1 and migrate from AWS SDK for Go v1 to v2.
- Replace the abandoned interactive shell dependency with `reeflective/readline`.
- Add stable `text`, `json`, and value-only output modes.
- Add command timeouts, cancellation, dry-run support, and secret-safe file and standard-input handling.
- Make copy and move preserve parameter metadata, policies, and tags.
- Add Linux, macOS, and Windows CI, dependency updates, vulnerability scanning, linting, and reproducible GoReleaser archives.

### Breaking changes

- Source builds require Go 1.27.1.
- `get` and `history` text output is now tab-separated. Use `-output value` when only parameter values are needed or `-output json` for structured output.
- Inline and batch commands return a nonzero status on AWS, validation, or parsing errors, and batch execution stops at the first failure.
- `get` fails when any requested parameter is missing instead of returning only the parameters that exist.
- Configuration parsing is strict. Only the documented keys in `[default]` are accepted, and a missing file passed with `-config` is an error.
- Leaving the profile unset uses the complete AWS SDK credential chain instead of forcing the `default` shared profile.
- Copy and move require `ssm:ListTagsForResource` and `ssm:AddTagsToResource` when tags are preserved.
- Cross-region SecureString copies require an explicit destination KMS key.
- The interactive prompt changed and the legacy `clear` command was removed.
- `make build` writes `bin/ssmsh`; use `make install` to install through the Go toolchain.
- Exported Go APIs now use contexts and AWS SDK for Go v2 types.

### Added

- `-output text|json|value`, `-timeout`, and `-dry-run` global options.
- `value-file` and `value-stdin` inputs for writing secrets without command-line exposure.
- AWS IAM Identity Center, web identity, and workload credential support through the AWS SDK v2 credential chain.
- Recursive mutation planning that rejects self-referential copy and move destinations.
- Deterministic ordering, pagination, and API-limit-aware batching.
- Unit coverage for configuration, AWS loading, command execution, output, path operations, and application exit behavior.
- Dependabot configuration and pinned CI/release tooling.

### Changed

- AWS clients are initialized lazily, allowing local help and shell startup without valid AWS credentials.
- Copy preserves descriptions, allowed patterns, data types, tiers, policies, KMS configuration, and tags.
- Move finishes all destination copies before deleting any source parameter.
- Remove validates the complete request before performing deletions.
- Ctrl-C cancels an in-flight operation instead of terminating without cleanup.
- Homebrew releases build from the published source archive.

### Fixed

- Single-word inline commands now execute instead of opening the interactive shell.
- Missing command arguments return usage errors instead of panicking.
- Permission and network failures are no longer mistaken for missing parameters.
- Parameter tiers accept `Standard`, `Advanced`, and `Intelligent-Tiering` values correctly.

[2.0.0]: https://github.com/bwhaley/ssmsh/compare/v1.4.9...v2.0.0
