# Security

lazybw is a terminal UI for the Bitwarden CLI (`bw`). While the vault is
unlocked, lazybw holds decrypted vault data in its memory. This document
describes how lazybw handles secrets, what it cannot protect, and the risks
that it accepts.

## Reporting a vulnerability

Do not report a vulnerability in a public issue. Other users can read it
before a fix is available. Report it in private with GitHub private
vulnerability reporting:
[github.com/juthrbog/lazybw/security/advisories/new](https://github.com/juthrbog/lazybw/security/advisories/new).

Put this information in the report:

- The output of `lazybw --version` and `bw --version`.
- Your operating system and terminal, and your terminal multiplexer if you
  use one.
- The steps that show the problem.

Do not put vault data, a master password or a session key in the report.
Steps with a test vault are enough.

## Secret lifecycle

### While the vault is unlocked

`bw list items` returns the whole vault, decrypted, as JSON. lazybw parses
the JSON into items and keeps them in the vault screen. The items hold
passwords, TOTP keys, notes, card numbers and security codes, SSNs,
passport and license numbers, and SSH private keys. lazybw also keeps the
session token (the `BW_SESSION` key) and the email of the account.

The JSON also has fields that lazybw does not parse, for example the
password history, custom field values and passkey keys. These values exist
only in the output of `bw`. That is the buffer that holds its stdout, and
the copy buffers of `os/exec` (see
[What stays in memory](#what-stays-in-memory)).

### Writes to the vault

lazybw can create and edit logins and secure notes. It runs
`bw create item` and `bw edit item`, which write the item to the Bitwarden
server and to the local data of `bw`. lazybw does not delete items yet.

The item goes to `bw` as base64-encoded JSON on stdin. It never goes in an
argument, because any local user can read the arguments of a process in
`ps` or `/proc/PID/cmdline`. lazybw zeroes the JSON and the encoded input
after the command. `bw create` prints the new item, decrypted. lazybw reads
only its ID and then zeroes that output.

A new item gets no organization, no folder and no master password
re-prompt. For an edit, lazybw runs `bw get item` and changes only the
fields that you changed. All other keys go back to `bw` as `bw get item`
printed them, for example the password history, custom fields and passkeys.
`bw get item` prints the decrypted item. lazybw zeroes that output after
it decodes it. The decoded strings stay in memory, as the strings of
`bw list items` do. The item form masks the password until you press `ctrl+r`, and
each new form masks it again.

If the vault locks while the item form shows, lazybw drops the form and its
values. lazybw keeps no draft of a form, in memory or on disk.

### What lazybw zeroes and drops

On a lock, an idle lock and a quit, lazybw does these actions:

- It zeroes the decoded TOTP key bytes (`totp.Params.Secret`) in place.
- It sets the secret string fields of each item to `""`: `Notes`,
  `Login.Password`, `Login.Totp`, `Card.Number`, `Card.Code`,
  `Identity.SSN`, `Identity.PassportNumber`, `Identity.LicenseNumber` and
  `SSHKey.PrivateKey`. The list rows share the `Login`, `Card`, `Identity`
  and `SSHKey` structs with the items, so this also covers the rows.
- It removes all list rows and clears the filter. Each row has its own copy
  of `Notes`.
- It drops the current TOTP code, the generated password, the text of an
  open master password prompt and the values of an open item form.
- It replaces the vault screen with an empty one.
- It sets the session token to `""` and runs `bw lock`. See
  [Lock and exit](#lock-and-exit).

After a sync, lazybw drops the secret fields of the old items in the same
way. It does the same for an item list that arrives after its session
ended.

lazybw also zeroes these byte buffers:

- The stdout of `bw` goes into a buffer that zeroes its old array each time
  it grows. So the buffer leaves no partial copy of the vault JSON behind.
- After the parse, lazybw zeroes the stdout of `bw list items`, `bw unlock`
  and `bw login`. When a `bw` command fails, lazybw zeroes its stdout.

When you submit the unlock or login form, lazybw takes the master password
and sets the string that the form writes to `""`. Then it puts a new, empty
form in place of the old one. After the unlock, lazybw drops the unlock
screen. A two-step login runs `bw login` again for each step. So lazybw
keeps the master password until the login succeeds or fails, or until you
press `esc`.

### What stays in memory

Go strings cannot be zeroed. When lazybw sets a field to `""`, it drops a
reference, but the bytes stay in the heap. The garbage collector frees that
memory later, but it does not zero it. The bytes stay until the Go runtime
uses the memory again. These copies stay in memory in this way:

- The strings of each item from the JSON parse, for example passwords,
  notes and card numbers.
- The master password in the old unlock form and in the old form of the
  master password re-prompt. The huh text input keeps it in a rune slice,
  and the huh form keeps a copy in its results. The public
  API of huh cannot zero them. The key press messages of Bubble Tea also
  carry each typed character.
- The master password and the session token in the environment strings of
  each `bw` child process (`BW_PASSWORD` and `BW_SESSION`).
- The copy buffers inside `os/exec` that move the stdout of `bw` into
  lazybw. lazybw cannot get to them.
- The output of `bw generate`. lazybw does not zero it.
- The values that you type in the item form. The bubbles text inputs keep
  them in rune slices, and the key press messages of Bubble Tea carry each
  typed character. The item JSON for `bw create item` holds them as Go
  strings until the garbage collector frees it.
- Each copied value, inside the OSC 52 sequence that Bubble Tea writes to
  the terminal.
- The fields that are not secret: item names, usernames, URLs, card holder
  names, identity names and addresses, and SSH public keys. lazybw does not
  blank them. They become garbage when lazybw replaces the vault screen.

The email of the account stays after a lock, because the unlock screen
shows it. `bw status` returns the email with no authentication, so lazybw
does not treat it as a secret.

Each `bw` process holds decrypted data while it runs. The operating system
frees that memory when the process exits. lazybw cannot zero the memory of
`bw`.

### TOTP keys

lazybw computes TOTP codes itself. When you select an item with a TOTP key,
lazybw parses the key and decodes the base32 secret into bytes. lazybw
zeroes these bytes when you select another item, and on a lock or a quit.
This TOTP data stays in memory until the Go runtime uses the memory again:

- The base32 key string in the item (`Login.Totp`). lazybw drops the
  reference on a lock, but it cannot zero a Go string.
- The string copies that the parse of the key makes. For example, the parse
  makes a lowercase copy of the whole key.
- The HMAC buffers. lazybw computes the code again each second while the
  item is selected. Each computation makes two buffers from the key (the
  HMAC inner and outer pads), and the Go HMAC code does not zero them.

## Lock and exit

### Exit paths

These actions clear the clipboard and lock `bw`:

- The `l` key (lock).
- The idle lock.
- `q`, then `y`, on the vault screen.
- `ctrl+c` on any screen.
- `q` on the error screen.
- `ctrl+q` on the unlock screen.
- SIGTERM, and SIGHUP, which the system sends when the terminal closes.
- A panic in `Update` or `View`.

lazybw clears the clipboard only when a value that it copied can still be
on it. It runs `bw lock` only when the session is open. On SIGTERM, SIGHUP
and a panic, the UI stops before lazybw can act. So lazybw does the cleanup
after the UI stops: it writes the clipboard clear to the terminal and runs
`bw lock`. During this cleanup, lazybw ignores SIGHUP, SIGTERM and SIGINT,
so that a closed terminal cannot stop the `bw lock` process. If an earlier
`bw lock` still runs at exit, the cleanup runs `bw lock` again with the same
token.

### Idle lock

lazybw locks the vault when no key press arrives for the `--idle-timeout`
time. The default is 15 minutes, and the minimum is 30 seconds. Only a key
press counts as activity. Every 30 seconds, lazybw compares the idle time
with the timeout. It measures the idle time with the wall clock as well as
the monotonic clock, so the time that the computer spends in suspend
counts. After the timeout, the first key press locks the vault and does not
count as activity.

### What `bw lock` does

`bw lock` locks the local session of the `bw` CLI. It does not end the
login. The login refresh token stays in the data of `bw` until
`bw logout`. lazybw never runs `bw logout`.

### When `bw lock` fails

lazybw drops the session token in every case, also when `bw lock` fails.
But `bw` can then still hold an unlocked session. So lazybw shows the error
of `bw lock` in these cases:

- After the `l` key or an idle lock, the unlock screen shows the error. An
  idle lock adds the error to the unlock screen that already shows, and
  keeps what you typed.
- After a second failed item load, lazybw locks the vault from the error
  screen. The next unlock screen shows the error. If you quit from the
  error screen, lazybw prints the error to stderr.
- After a quit, lazybw prints the error to stderr after it restores the
  terminal.
- The exit cleanup runs `bw lock` when the session is still open at
  SIGTERM, SIGHUP or a panic. It also runs `bw lock` when an earlier
  `bw lock` did not return before the exit. lazybw prints the error of this
  `bw lock` to stderr too.

lazybw shows no error when `bw` reports that no account is logged in,
because the old session key then opens nothing. If you unlock again before
the result of `bw lock` arrives, the error goes only to the debug log. To
end a session that `bw` still holds, run `bw lock` or `bw logout` in a
terminal.

## Master password handling

lazybw passes the master password to `bw` with `--passwordenv BW_PASSWORD`.
`bw` then reads the password from the `BW_PASSWORD` variable in its own
environment. lazybw sets this variable only in the environment of the `bw`
child process, not in its own environment. So the password does not show in
`ps`, `top` or `/proc/PID/cmdline`. While `bw` runs, the owner of the
process and root can read the password in `/proc/PID/environ` of the `bw`
process.

lazybw does not test the version of `bw` at start.

## Master password re-prompt

In Bitwarden, you can turn on master password re-prompt for an item. `bw`
gives this setting to lazybw but does not enforce it. lazybw enforces it.
For such an item, `c`, `t`, `v` and `m` ask for the master password before
they copy, and `e` asks before it opens the edit form, which shows each
field. The drawer hides the TOTP code and the note text until you enter
the password.

`u` (username) and `o` (open URL) do not ask. After the correct password,
lazybw does not ask again while the item stays selected. A new selection, a
lock, or a sync that removes the item ends this.

Re-prompt is a UI control, not encryption. `bw list items` returns these
items decrypted, as it does all other items, and lazybw holds them in
memory in the same way. To test the password, lazybw runs `bw unlock` with
it. When `bw unlock` succeeds, `bw` returns a new session key, and the old
key stops working. lazybw then uses the new key. A wrong password copies
nothing and shows an error.

## Session token and the `bw` environment

lazybw keeps the session token in memory only. It passes the token to each
`bw` child process in the `BW_SESSION` environment variable. This is the
standard mechanism of the Bitwarden CLI.

Each `bw` child process gets the environment of lazybw without these
variables: `BW_SESSION`, `BW_PASSWORD`, `BW_RESPONSE`, `BW_QUIET`,
`BW_RAW`, `BW_PRETTY`, `BW_CLEANEXIT`, `BW_NOINTERACTION` and
`BITWARDENCLI_DEBUG`. Because lazybw removes `BW_SESSION`, it never uses a
session key from the shell that started it. The other `BW_` variables
change what `bw` prints. `BITWARDENCLI_DEBUG` makes `bw` write its debug log
to stdout, in front of the JSON and the session key.

Then lazybw adds `BW_NOINTERACTION=true`, so that `bw` fails at once when it
needs an answer that nobody can type. It adds `BW_SESSION` when a session is
open, and `BW_PASSWORD` for an unlock or a login. The other variables of
lazybw stay, for example `BITWARDENCLI_APPDATA_DIR`, `NODE_EXTRA_CA_CERTS`
and the proxy variables.

## Clipboard

lazybw copies only through OSC 52, an escape sequence that asks the
terminal to set the clipboard. lazybw starts no clipboard program. The
terminal sends no reply, so lazybw cannot tell if a copy worked. For this
reason, the copy message says that lazybw sent the value to the clipboard.
It does not say that the copy worked.

OSC 52 needs support in the terminal. In tmux, it also needs
`set -g set-clipboard on`. With the default value, tmux drops the copy and
gives no sign.

lazybw clears the clipboard 60 seconds after the last copy. Each copy
starts a new timer, and only the timer of the latest copy clears the
clipboard. lazybw also clears the clipboard on each exit path in
[Lock and exit](#lock-and-exit). The clear writes a one-space placeholder
(`" "`), not an empty value, because tmux ignores an OSC 52 write with an
empty payload. If lazybw copied nothing since the last clear, it does not
write to the clipboard.

## Debug log

With `--debug`, lazybw adds log lines to `$XDG_CACHE_HOME/lazybw/debug.log`.
If `XDG_CACHE_HOME` is not set, the file is `~/.cache/lazybw/debug.log`.
lazybw creates the directory with mode 0700 and the file with mode 0600.
Without `--debug`, lazybw writes no log.

The log has a line for each state change:

- The `bw status` result, with the status value only.
- A successful unlock.
- The item count after a load.
- The start of each lock: the lock key, the idle lock, a quit, or the lock
  after a second failed item load.
- The `bw lock` result.
- Sync results and errors.

The lines that lazybw writes never contain the session token, the master
password, the email, item names or field values. A line about an error
names only the kind of error, for example `timed out` or
`bw error (exit status 1)`. It does not repeat the message of `bw`. For a
failed `bw` command, the log also has the stderr of `bw`. lazybw replaces
the token, the password, the two-step code and the email in it with
`[hidden]`. For `bw create item` and `bw edit item`, it also replaces the
encoded item and each value of the item form.

## Bubble Tea trace file

Bubble Tea, the UI library of lazybw, writes a trace file when the
`TEA_TRACE` environment variable names a path. The trace records every key
press, which includes each character of the master password. It also
records the OSC 52 payloads, which hold the copied secrets in base64. So if
`TEA_TRACE` is set, lazybw prints a warning to stderr and removes the
variable before it starts the UI. Bubble Tea then writes no trace file.

## Memory locking (mlock)

lazybw does not call `mlock` or `mlockall`. The Go runtime allocates,
moves and copies memory on its own, for example when a goroutine stack
grows or shrinks. An `mlock` on some buffers does not stop the runtime from
putting copies of secrets in other pages. Libraries such as `memguard` add
complexity and give no reliable protection in a standard Go program.

## Accepted risks

- Under memory pressure, the system can write pages with secrets to swap.
  Most Go programs that handle secrets share this limit.
- A process that can read the memory of lazybw can find the copies in
  [What stays in memory](#what-stays-in-memory). Examples are a process of
  the same user with ptrace access, root, a core dump and swap.
- tmux keeps each copy that it accepts as a paste buffer. Other processes
  of the same user can read these buffers. lazybw does not delete them.
- A clipboard history manager keeps copies after the 60-second clear.
- A terminal or a multiplexer can drop an OSC 52 write and give no sign.
  Then a copy does not arrive, or a clear does not arrive and the secret
  stays on the clipboard.
- A SIGKILL or a power loss stops lazybw before any cleanup. Then the
  clipboard keeps the last copy, and `bw lock` does not run.

## Other areas

These areas have no known open problem:

- Shell injection: lazybw passes all arguments of `bw` and of the URL
  opener as `exec.Command` arguments, never through a shell. The email
  comes after `--`, so `bw` cannot read it as an option. Item data goes on
  stdin, never in an argument.
- URLs: `o` opens only `http` and `https` URLs.
- Terminal escape sequences: lazybw removes escape sequences and control
  characters from vault text before it draws the text. The copy keys copy
  the stored value.
