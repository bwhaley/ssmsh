# ssmsh

`ssmsh` is an interactive shell and command-line client for AWS Systems Manager Parameter Store. It provides familiar commands such as `cd`, `ls`, `get`, `put`, `cp`, `mv`, and `rm`, including relative paths, recursion, multiple regions, parameter history, advanced parameter policies, and batch execution.

The interactive prompt shows the active AWS profile, region, and Parameter Store path:

```text
[production@us-west-2] /service/api> ls
database/
endpoint
```

## Installation

Download macOS, Linux, or Windows archives from [GitHub Releases](https://github.com/bwhaley/ssmsh/releases), or install the current source with Go 1.27.1:

```bash
go install github.com/bwhaley/ssmsh@latest
```

The repository also acts as a Homebrew tap for macOS and Linux:

```bash
brew tap bwhaley/ssmsh https://github.com/bwhaley/ssmsh
brew install ssmsh
```

A community-maintained [Nix package](https://search.nixos.org/packages?channel=unstable&show=ssmsh&query=ssmsh) is also available.

### Upgrading from v1

Version 2 updates the CLI behavior as well as its dependencies. Before upgrading automation or shared environments, account for these changes:

- Source builds require Go 1.27.1.
- `get` and `history` use stable tab-separated text output. Use `-output value` for values alone or `-output json` for structured records.
- Command failures now produce a nonzero exit status, batch files stop at the first failure, and `get` fails if any requested name is missing.
- Configuration files accept only the documented keys in `[default]`; an explicitly selected missing file is an error.
- An unset profile uses the complete AWS SDK credential chain instead of selecting the `default` profile explicitly.
- Copying preserves tags and therefore needs `ssm:ListTagsForResource` and `ssm:AddTagsToResource`. Cross-region SecureString copies need a destination KMS key.
- Consumers of the Go packages must migrate to context-aware APIs and AWS SDK for Go v2 types.

See the [v2.0.0 changelog](CHANGELOG.md#200---2026-09-24) for the complete release notes.

## AWS configuration

`ssmsh` uses the AWS SDK for Go v2 credential chain. Environment credentials, shared profiles, IAM roles, web identity credentials, workload credentials, and IAM Identity Center (SSO) profiles are supported. Configure credentials using the [AWS SDK and Tools reference guide](https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html).

An optional `~/.ssmshrc` sets command defaults:

```ini
[default]
type=SecureString
overwrite=false
decrypt=false
profile=my-profile
region=us-east-1
key=alias/parameter-store
output=text
```

Use `-config path` to select another file. A missing explicitly selected file is an error; a missing default `~/.ssmshrc` is allowed.

`AWS_PROFILE` takes precedence over the configured profile. `AWS_REGION` takes precedence over the configured region. When neither is set, the AWS SDK resolves its usual environment, shared-config, and workload defaults. An empty profile leaves the complete SDK credential chain available.

Supported output modes are `text`, `json`, and `value`. Set one in the configuration file or with `-output`. JSON uses records owned by `ssmsh`, so SDK upgrades do not silently change the output field set.

## Usage

Run `ssmsh` for the interactive shell, pass a command directly, or read commands from a file:

```bash
ssmsh
ssmsh get /service/api/endpoint
ssmsh -output value get /service/api/password
ssmsh -file commands.txt
cat commands.txt | ssmsh -file -
```

Every command has a two-minute timeout by default. Change it with a Go duration such as `-timeout 30s`. Ctrl-C cancels an in-flight command.

Use `help` or `help COMMAND` to inspect available commands:

```text
cd           change parameter directory
cp           copy parameters
decrypt      set parameter decryption
exit         exit the interactive shell
get          get parameters
history      get parameter history
key          set the destination KMS key
ls           list parameters
mv           move parameters
policy       create a named parameter policy
profile      switch AWS profile
put          set a parameter
region       switch AWS region
rm           remove parameters
```

### Paths and regions

Paths may be absolute or relative to the current Parameter Store directory. Prefix a path with a region to target another region:

```text
cd /service
ls api
get api/endpoint
get us-east-1:/service/api/endpoint us-west-2:/service/api/endpoint
ls -r eu-central-1:/service
```

Use `profile NAME` and `region NAME` to switch the active AWS configuration. The new configuration is validated before the current client cache is replaced.

### Reading values and history

```bash
ssmsh get /service/api/endpoint
ssmsh -output json history /service/api/endpoint
ssmsh -output value get /service/api/password
```

SecureString values are encrypted in responses unless decryption is enabled:

```text
decrypt true
decrypt false
```

### Creating parameters

Supply a value directly, read it byte-for-byte from a file, or read it from standard input:

```bash
ssmsh put name=/service/api/endpoint value=https://example.com type=String
ssmsh put name=/service/api/private-key value-file=private.pem type=SecureString
printf '%s' "$SECRET" | ssmsh put name=/service/api/password value-stdin=true type=SecureString
```

`value-file` and `value-stdin` preserve whitespace and newlines. `value-stdin` cannot be combined with `-file -`, because both would consume the same stream. Inline values may be visible in shell history or process listings; prefer a file or standard input for secrets.

Interactive `put` with no arguments reads one `name=value` option per line until an empty line. Interactive command history remains in memory and is not written to disk.

Advanced parameter policies can be named and reused during a session:

```text
policy expiry Expiration(Timestamp=2030-01-01T00:00:00Z)
policy notices ExpirationNotification(Before=30,Unit=days) NoChangeNotification(After=7,Unit=days)
put name=/service/api/token value-file=token type=SecureString policies=[expiry,notices]
```

### Copying, moving, and removing

```text
cp /service/api/endpoint /backup/endpoint
cp -r /service /backup
mv /service/old-name /service/new-name
rm /service/api/endpoint
rm -r /service/obsolete
```

Copy preserves the current value, description, allowed pattern, data type, tier, policies, and tags. A cross-region SecureString copy requires a destination KMS key:

```text
cp key=alias/destination -r us-east-1:/service us-west-2:/service
```

You can also set the active destination key with `key KEY_ID_OR_ALIAS`.

`mv` copies every destination before deleting any source. If copying fails, sources remain. If source cleanup fails after a successful copy, the error identifies that the destinations exist and cleanup is incomplete.

Preview mutations with the global option or a command option:

```bash
ssmsh -dry-run rm -r /service/obsolete
ssmsh cp --dry-run -r /service /backup
ssmsh mv --dry-run /service/old /service/new
```

Dry-run output contains names and regions, never parameter values.

### Batch behavior

Batch files support blank lines, comments beginning with `#`, and shell-style quoting. Execution stops at the first invalid command or AWS error and returns a nonzero exit status. Error messages identify the line number without echoing the command, which could contain a secret.

```text
# commands.txt
put name=/service/api/endpoint value="https://example.com" type=String
cp /service/api/endpoint /backup/api/endpoint
rm /service/api/old-endpoint
```

## Development

The project requires Go 1.27.1. Common checks are exposed through the Makefile:

```bash
make check          # format, vet, and race-enabled tests
make lint           # static analysis
make vuln           # reachable vulnerability analysis
make build          # bin/ssmsh
make snapshot       # local GoReleaser snapshot
```

CI runs tests on Linux, macOS, and Windows, checks the module files, runs vulnerability analysis, and builds a release snapshot. Tagged releases use the pinned GoReleaser version and refresh the source-based Homebrew formula.

## License

MIT
