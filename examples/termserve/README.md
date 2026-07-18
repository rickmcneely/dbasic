# TermServe — host a DBasic **TermApp** over SSH

A **TermApp** is a normal DBasic terminal program (a Charm bubbletea app) that
runs on **one server** and is used remotely over SSH. Everyone gets the current
version (the binary lives in one place); a visitor needs nothing but an SSH
client and a terminal.

`termserve` is the host — itself **written in DBasic**. For each SSH connection it
checks the visitor's credentials, gives the TermApp a PTY, launches it, and
relays the session (keystrokes in, screen out, plus window resizes). It execs
the compiled binary unchanged, so it can host **any** DBasic TUI (DBtui, edit,
StarWord, …).

The design rationale and roadmap (in-process hosting, web/browser delivery, a
future custom client) are in [`docs/REMOTE_SERVING.md`](../../docs/REMOTE_SERVING.md).

## Authentication is required

Because a TermApp runs **on the server**, `termserve` never allows anonymous
access. It **refuses to start** unless you configure at least one way to log in:

- **SSH keys** — put the allowed public keys in an `authorized_keys` file
  (one per line, standard OpenSSH format):
  ```bash
  cp ~/.ssh/id_ed25519.pub  authorized_keys      # or append several
  ```
- **Password** — set it in the environment (never hard-code it):
  ```bash
  export TERMSERVE_PASSWORD='choose-a-good-one'
  ```

You can enable either or both. Passwords are compared in constant time
(`crypto/subtle`). The `authorized_keys` file and any keypairs are gitignored so
you can't accidentally commit credentials.

## Build

```bash
cd examples/termserve
dbasic build termserve.dbas -o termserve
dbasic build demo/hello_tui.dbas -o demo/hello_tui   # the demo TermApp to host
```

The first build fetches `charmbracelet/wish`, `charmbracelet/ssh`, and
`creack/pty`. After that, `dbasic build --offline` works.

## Run

```bash
cp ~/.ssh/id_ed25519.pub authorized_keys     # allow your key
./termserve demo/hello_tui                      # listen on :2222
```

Then connect from anywhere:

```bash
ssh -p 2222 your-server
```

You'll see the TermApp; resize your terminal and it repaints; press `q` to quit.

Environment knobs:

| Variable | Meaning | Default |
| --- | --- | --- |
| `TERMSERVE_ADDR` | listen address | `:2222` |
| `TERMSERVE_AUTHORIZED_KEYS` | path to the allowed-keys file | `authorized_keys` |
| `TERMSERVE_PASSWORD` | enable password login | *(off)* |

## Verify (automated)

```bash
./test/run_verify.sh
```

The harness runs entirely in a scratch dir and checks, end-to-end:

1. `termserve` **refuses to start** with no authentication configured;
2. an **SSH-key** login renders the TermApp and a **live resize** propagates;
3. a **password** login works (needs `sshpass`);
4. a key **not** in `authorized_keys` is **rejected**.

Requires `ssh`, `ssh-keygen`, and `python3` with `pyte` (`pip install pyte`);
`sshpass` is optional (test 3 skips without it).

## How it works (code map)

`termserve.dbas`:

- `Main` — reads settings, **locks the door** (refuses to start without keys or a
  password), builds the option list, and starts the wish server. Options are
  gathered into a `[]ssh.Option` slice and spread into `wish.NewServer(opts...)`.
- `passwordCheck` — constant-time password comparison for `wish.WithPasswordAuth`.
  SSH-key logins go through `wish.WithAuthorizedKeys`.
- `serveSession` — per-visitor handler: allocate the PTY (`s.Pty()`), start the
  TermApp (`pty.Start`), copy visitor→PTY and PTY→visitor.
- `watchResize` — forwards window-change events. `s.Pty()` returns a receive-only
  channel (`RECEIVE CHAN OF ssh.Window`), drained with the comma-ok
  `RECEIVE win, ok FROM winCh`.
- `appMiddleware` — plugs the handler into wish, wrapped by `logging` + `activeterm`.

## ⚠️ Remember: the TermApp runs on the SERVER

All of a hosted TermApp's local I/O — files, audio, clipboard, sub-processes —
happens **server-side**, not on the visitor's machine. The demo app writes
`termserve_proof.txt` on startup; notice it lands in the **server's** working
directory. A multi-user TermApp must key its per-user state by session identity
(SSH username / public key), not a shared home directory. (Ebitengine games are
graphical, not VT100, so they're out of scope.) See
[`docs/REMOTE_SERVING.md`](../../docs/REMOTE_SERVING.md) for the full trust model.
