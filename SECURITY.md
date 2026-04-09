# Security Considerations

lazybw is a terminal UI wrapper around the Bitwarden CLI (`bw`). It
necessarily holds decrypted vault data in memory while the session is
unlocked. This document describes the security design, mitigations, and
accepted risks.

## Secret lifecycle

When the vault is unlocked, decrypted items (passwords, TOTP seeds, card
numbers, SSNs, SSH private keys) are cached in the `VaultModel`. On lock
(manual, idle timeout, or quit):

1. **Byte-slice secrets are zeroed in-place.** The decoded TOTP secret
   (`totp.Params.Secret`) is overwritten with zeros before the reference
   is dropped. This genuinely scrubs the key material from memory.

2. **String-field secrets have references dropped.** Go strings are
   immutable — the backing bytes cannot be zeroed from safe Go code. We
   set each sensitive string field to `""`, which releases the reference
   and allows the garbage collector to reclaim the underlying memory. The
   old bytes may persist in the heap until the GC runs.

3. **The session token is cleared.** `State.Lock()` sets `Token` to `""`
   and resets `LastSync`. A `bw lock` command is also issued to the CLI
   to lock the server-side session.

4. **Email is retained.** The user's email is kept after lock for display
   on the unlock screen. It is not considered secret — `bw status`
   returns it without authentication.

## Master password handling

The master password is passed to the `bw` CLI via `--passwordenv
BW_PASSWORD`, which reads it from a process-local environment variable.
This avoids exposing the password in `ps`, `top`, or
`/proc/PID/cmdline`. The environment variable is only readable by the
process owner (or root) via `/proc/PID/environ`.

Minimum `bw` CLI version required: **1.21.0** (when `--passwordenv` was
introduced).

## Session token

The `BW_SESSION` token is passed to child `bw` processes via the
`BW_SESSION` environment variable. This is the standard mechanism used by
the Bitwarden CLI itself.

## Clipboard

Clipboard operations use OSC 52 terminal escape sequences — no
intermediate clipboard manager processes are spawned. Copied secrets are
automatically cleared after 60 seconds.

## Memory locking (mlock)

lazybw does **not** call `mlock` or `mlockall`. Go's runtime freely
allocates, moves, and copies memory (GC compaction, goroutine stack
growth/shrinking). Calling `mlock` on specific buffers would not prevent
the runtime from placing copies of sensitive data in unlocked pages.
Libraries like `memguard` add complexity without reliable guarantees in a
standard Go program.

**Accepted risk:** Under memory pressure, pages containing secret data
could be swapped to disk. This is a known limitation shared by most Go
applications that handle secrets.

## Subprocess environment

Child `bw` processes inherit the parent's full environment via
`os.Environ()`, with `BW_SESSION` (and `BW_PASSWORD` for login/unlock)
appended. This is standard practice and ensures the child process has
access to necessary system configuration (locale, PATH, etc.).

## Out of scope

These areas were reviewed and found to be already well-handled:

- **Clipboard**: OSC 52 with 60-second auto-clear
- **Debug logging**: No secrets logged; log files use 0600 permissions
- **Shell injection**: All subprocess arguments are passed as
  `exec.Command` argv, never through a shell
