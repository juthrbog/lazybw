# Install lazybw

This page has the details of each install method. For the short version, see the [README](../README.md#installation).

lazybw needs the [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/#download-and-install) on your `$PATH`. lazybw is one binary, and at run time it needs only `bw`.

## Homebrew

On macOS or Linux:

```sh
brew install bitwarden-cli juthrbog/tap/lazybw
```

The `juthrbog/tap/lazybw` formula installs the release binary. It does not install `bw` by itself, so the command above also installs `bitwarden-cli`. The tap has no Windows entry, because Homebrew does not run on Windows.

## Download a binary

1. Download the archive for your system from the [releases page](https://github.com/juthrbog/lazybw/releases/latest). Releases have binaries for Linux, macOS and Windows, on amd64 and arm64. The Windows archives are zip files, and the others are tar.gz files.
2. Extract the archive.
3. Put `lazybw` (`lazybw.exe` on Windows) in a directory on your `$PATH`, for example `/usr/local/bin`.
4. Install `bw`, if you do not have it.

## Verify a download

Each release also has these files:

- `checksums.txt`: the SHA-256 checksum of each archive and each SBOM.
- `checksums.txt.sigstore.json`: a cosign signature of `checksums.txt`.
- One SBOM for each archive, in SPDX JSON. An SBOM (software bill of materials) lists the Go modules in the binary.

To make sure that an archive comes from the release workflow of this repository, use one of the two methods below. In each command, replace `<version>` with the version of the release without the `v` (for example `0.5.0`), and `linux_amd64` with your platform.

### With the GitHub CLI

This command uses the build provenance attestation that GitHub keeps for each release file:

```sh
gh attestation verify lazybw_<version>_linux_amd64.tar.gz --repo juthrbog/lazybw
```

### With cosign

1. Download `checksums.txt` and `checksums.txt.sigstore.json` from the same release, into the directory of the archive.
2. Make sure that the release workflow signed `checksums.txt`:

   ```sh
   cosign verify-blob checksums.txt \
     --bundle checksums.txt.sigstore.json \
     --certificate-identity-regexp '^https://github\.com/juthrbog/lazybw/\.github/workflows/release\.yml@refs/tags/v' \
     --certificate-oidc-issuer https://token.actions.githubusercontent.com
   ```

3. Compare the checksum of the archive with `checksums.txt`:

   ```sh
   sha256sum --ignore-missing -c checksums.txt
   ```

## With Go

This needs Go 1.26.0 or later, which is the `go` line in `go.mod`. It does not install `bw`.

```sh
go install github.com/juthrbog/lazybw@latest
```

If your Go is older and `GOTOOLCHAIN` has its default value `auto`, the `go` command downloads a newer Go first.

## Build from a clone

```sh
git clone https://github.com/juthrbog/lazybw.git
cd lazybw
go build -o lazybw .
```

A build in a clone also follows the `toolchain go1.27.2` line in `go.mod`, so it uses Go 1.27.2 or later.

## Version output

`lazybw --version` prints the version and exits. The format depends on how you installed lazybw:

| Build | Output |
|---|---|
| Release binary (Homebrew or download) | The release version without a `v`, for example `lazybw 0.5.0` |
| `go install` | The module version, for example `lazybw v0.5.0` |
| Build from a clone | The git tag or a pseudo-version, for example `lazybw v0.5.0` or `lazybw v0.5.1-0.20261101...+dirty`. lazybw adds `+dirty` if files in the clone changed. |
| A build with no git information, for example from a source archive | `lazybw dev` |

[DESIGN.md](DESIGN.md#command-line-flags) explains how lazybw gets the version.
