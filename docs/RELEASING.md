# Releasing lazybw

A tag that starts with `v` publishes a release. `.github/workflows/release.yml` builds the archives with GoReleaser, signs them, and updates the Homebrew formula.

## Make a release

1. Make sure that CI passes on `main`.
2. Switch to `main`, and pull.
3. Run the release task with the new version:

   ```sh
   task release -- v0.3.0
   ```

   The task stops if the version does not look like `v1.2.3` or `v1.2.3-rc.1`, if the branch is not `main`, if the tree has changes, or if `main` is not the same as `origin/main`. Then it pushes the tag.
4. Watch the Release workflow on the Actions tab of the repository.

A version with a hyphen, such as `v0.3.0-rc.1`, makes a prerelease. A prerelease does not change the Homebrew formula.

## What a tag publishes

The `test` job runs first. It runs `go vet ./...`, `go test -race -shuffle=on ./...` and `govulncheck ./...`. It has no write permission. A failure stops the release.

Then the `release` job runs GoReleaser with `.goreleaser.yml`:

- Archives for Linux, macOS and Windows, each on amd64 and arm64. The Windows archives are zip files. The others are tar.gz files. Each archive has the binary, `README.md` and `LICENSE`.
- One SBOM for each archive, in SPDX JSON. syft writes them.
- `checksums.txt`, with the SHA-256 checksum of each archive and each SBOM.
- `checksums.txt.sigstore.json`, a keyless cosign signature of `checksums.txt`. The signature holds the identity of the release workflow.
- Release notes that GitHub writes from the merged pull requests. `.github/release.yml` puts the Dependabot updates in a section of their own.

The builds use `-trimpath`, and the file times come from the commit. Two builds of the same commit give the same archives. The SBOMs differ, because each one holds its creation time and a random ID.

After GoReleaser, the job adds a build provenance attestation for each file in `checksums.txt`. Then it writes `Formula/lazybw.rb` with `scripts/formula.sh` and pushes it to [juthrbog/homebrew-tap](https://github.com/juthrbog/homebrew-tap). The formula has the URL and the checksum of the archive for each platform. It has no Windows entry, because Homebrew does not run on Windows.

The README shows how users check an archive with `gh attestation verify` or `cosign verify-blob`.

## Test the release setup

A pull request that changes `.goreleaser.yml`, `.github/release.yml`, `.github/workflows/release.yml` or `scripts/formula.sh` runs two other jobs. You can also start them by hand on the Actions tab (`workflow_dispatch`).

- `snapshot` runs GoReleaser with `--snapshot --skip=sign`. It builds every archive and every SBOM, writes the formula, and checks it with `ruby -c`. It publishes nothing. Keyless signing works only in a tag build.
- `verify` runs `lazybw --version` from the archive on Linux amd64, Linux arm64, macOS arm64 and Windows amd64.

To build the archives on your computer, run `task release-dry-run`. It skips the signature and the SBOMs, so it needs only GoReleaser. The archives go to `dist/`.

## The Homebrew deploy key

The release workflow pushes to juthrbog/homebrew-tap with a deploy key. The key has write access to that repository only. Its title in the tap is "lazybw release publisher". Its private key is the `HOMEBREW_TAP_SSH_KEY` Actions secret of this repository. No repository holds the private key.

If the secret is missing, the release still publishes the archives. It skips the formula and shows a warning.

To replace the key:

1. Make a new key with no passphrase: `ssh-keygen -t ed25519 -N "" -C "lazybw release publisher" -f lazybw_tap`.
2. Add the public key to the tap with write access: `gh repo deploy-key add lazybw_tap.pub --repo juthrbog/homebrew-tap --title "lazybw release publisher" --allow-write`.
3. Set the secret: `gh secret set HOMEBREW_TAP_SSH_KEY --repo juthrbog/lazybw < lazybw_tap`.
4. Delete the old deploy key in the tap under Settings, Deploy keys.
5. Delete the two key files from your computer.
