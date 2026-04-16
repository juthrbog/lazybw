# Local Development Environment

A local [Vaultwarden](https://github.com/dani-garcia/vaultwarden) instance provides an isolated, reproducible environment for development and testing without needing internet or a production vault.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2
- [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/#download-and-install) in `$PATH`
- [Task](https://taskfile.dev/installation/) runner
- Go 1.22+

## Quick Start

```sh
task dev:seed   # starts Vaultwarden, creates test account, seeds vault
task dev:run    # run lazybw against the local instance
```

## Test Account

| Field    | Value                      |
|----------|----------------------------|
| Server   | https://localhost:8443     |
| Email    | test@lazybw.dev            |
| Password | master-password-for-dev    |

The Vaultwarden admin panel is available at https://localhost:8443/admin with token `lazybw-dev-admin-token` (accept the self-signed certificate warning).

## Seeded Items

The seed tool creates 9 items covering all 5 supported item types. Several items share base names to exercise the grouping feature (Ctrl+G):

| # | Type         | Name              | Key Fields                                     | Group     |
|---|--------------|-------------------|-------------------------------------------------|-----------|
| 1 | Login        | GitHub (Personal) | Username, password, TOTP, URL                   | github    |
| 2 | Login        | GitHub (Work)     | Username, password, TOTP, URL                   | github    |
| 3 | Login        | AWS (Production)  | Username, password, URL (no TOTP)               | aws       |
| 4 | Login        | AWS (Staging)     | Username, password, URL                         | aws       |
| 5 | Secure Note  | AWS Access Key    | Access key ID + secret (absorbed into AWS group)| aws       |
| 6 | Secure Note  | API Keys          | Multi-line content                              | -         |
| 7 | Card         | Visa Debit        | Cardholder, number (4111...1111), expiry, CVV   | -         |
| 8 | Identity     | Personal Identity | All fields: name, email, SSN, address, etc.     | -         |
| 9 | SSH Key      | Dev SSH Key       | Ed25519 keypair (generated at seed time)        | -         |

## Tasks Reference

| Task          | Description                                          |
|---------------|------------------------------------------------------|
| `dev:up`      | Start Vaultwarden (waits for healthcheck)            |
| `dev:seed`    | Start Vaultwarden + create account + seed items      |
| `dev:down`    | Stop Vaultwarden                                     |
| `dev:reset`   | Destroy all data and re-seed from scratch            |

## How It Works

The seed tool (`cmd/vwseed/`) handles the full setup:

1. Creates `.bw-dev/` directory for isolated `bw` CLI config (via `BITWARDENCLI_APPDATA_DIR`)
2. Waits for Vaultwarden's `/alive` health endpoint
3. Registers an account via `/api/accounts/register` (requires PBKDF2 key derivation and RSA keypair generation -- the `bw` CLI cannot create accounts)
4. Configures `bw` CLI to point at localhost:8080
5. Logs in and creates all seed items via `bw create item`
6. Locks the vault (so lazybw starts at the unlock screen)

The seed tool is idempotent: running it again skips registration (HTTP 409) and item creation (vault already populated).

## Config Isolation

The dev environment uses `BITWARDENCLI_APPDATA_DIR=.bw-dev/` to keep its `bw` CLI config completely separate from your production Bitwarden config at `~/.config/Bitwarden CLI/`. This means:

- `task dev:seed` and `task dev:run` never touch your production config
- Running `lazybw` normally (without the env var) still uses your real vault
- `task dev:reset` cleans up `.bw-dev/` along with the Docker volume

## Troubleshooting

### Port 8443 already in use

Stop the conflicting service or edit `compose.yaml` to use a different port. Update the `serverURL` constant in `cmd/vwseed/main.go` to match.

### `bw` CLI version issues

`bw` CLI 2025.12.0+ has [compatibility issues](https://github.com/dani-garcia/vaultwarden/issues/6729) with Vaultwarden when using API key auth. The seed tool uses email/password login, which is unaffected. If you encounter issues, pin to `bw` 2025.11.0.

### SSH key items

SSH key vault items require the `ssh-key-vault-item` experimental feature flag, which is enabled in `compose.yaml`. If the `bw` CLI lacks a template for type 5 items, the seed tool constructs the JSON manually.

### "Account already exists"

This is normal on repeated runs. The seed tool handles HTTP 409 gracefully.

### Starting fresh

```sh
task dev:reset
```

This runs `docker compose down -v` (destroys the volume) then re-seeds.

### Running lazybw against local without `task dev:run`

Set the env var manually:

```sh
BITWARDENCLI_APPDATA_DIR=.bw-dev go run .
```

Without this env var, lazybw uses your normal `bw` CLI config (production vault).
