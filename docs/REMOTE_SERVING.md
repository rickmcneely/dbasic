# Remote serving of DBasic TUIs — "TermApps"

Run a DBasic terminal app on **one server** and let many people connect to it
from anywhere. Everyone runs the current version (the binary lives in one
place), data and secrets stay central, there is nothing to install on the
client beyond a terminal, and it works on every OS for free. We call an app used
this way — a DBasic TUI hosted on a server and reached over SSH — a **TermApp**.

This document explains why that works, the delivery options from cheapest to
richest, the path to a custom client, and the caveats you must design around.
A working, authenticated implementation lives in
[`examples/termserve/`](../examples/termserve/).

## Why it works

Every DBasic TUI is a [Charm](https://github.com/charmbracelet) **bubbletea**
program. A bubbletea program is pure **stdin → stdout**:

- it reads keystrokes (and mouse/resize events) from stdin, and
- it writes ANSI / VT100 escape sequences to stdout to paint the screen.

That byte stream is exactly what a terminal carries over a wire. Nothing about
the app assumes it is running on the same machine as the human. So you can put
the app on a server, connect the app's stdin/stdout to a network session, and
the user drives it remotely — the same way `ssh`-ing into a box and running a
full-screen program has always worked.

Bandwidth stays low because bubbletea renders **diffs**: it repaints only the
cells that changed between frames, not the whole screen every tick. Interactive
TUIs feel fine over ordinary links.

## Delivery tiers

### Tier 0 — SSH forced-command (works today, zero code)

You can serve any existing DBasic binary right now with no toolchain change.
On a server, pin an SSH key to run the binary instead of a shell:

```
# ~serve/.ssh/authorized_keys
command="/srv/apps/myapp",no-port-forwarding,no-X11-forwarding ssh-ed25519 AAAA... user@host
```

The user runs `ssh serve@host` and lands directly in the app; `sshd` allocates
the PTY and handles resize. Good for a quick internal deployment. Downsides:
you manage OS users and `authorized_keys` by hand, and everyone shares one
server account unless you split them.

### Tier 1 — a `wish` SSH server (the recommended path, and the PoC)

[`charmbracelet/wish`](https://github.com/charmbracelet/wish) is Charm's own SSH
server framework, purpose-built to serve bubbletea apps. It handles public-key
auth, PTY allocation, window-resize, and concurrent sessions, and it is how
Charm hosts public apps like `ssh terminal.shop` and `ssh chess.rest`.

There are two ways to wire a DBasic app into wish:

- **exec-in-PTY (app-agnostic).** The wish handler spawns the *already-compiled*
  DBasic binary attached to a PTY and copies bytes both ways. It serves **any**
  DBasic TUI unchanged — no edits to the app. This is what
  [`examples/termserve/`](../examples/termserve/) implements, in DBasic. It is the
  simplest thing that fully works and it is transport-agnostic, so the same
  server can later speak a custom protocol (see below).

- **in-process model factory (native, richer).** wish's `bubbletea.Middleware`
  runs a bubbletea model *inside the server process*, one fresh model per
  session, with the SSH session as the model's input/output. This is more
  efficient (no child process per connection) and gives the server direct
  access to per-session state. DBasic already has the natural seam for it:
  `SUB Main()` in a generated app builds the model with a factory such as
  `newModel()` and hands it to `tea.NewProgram(...)`
  (see `examples/DBtui/dbtui.dbas`). A future `dbasic build --serve` target
  could emit the wish wrapper automatically around that factory. **Not built
  yet** — documented here as the next step.

The PoC uses exec-in-PTY because it serves existing binaries with no changes
and keeps the transport swappable.

### Tier 2 — web / browser (zero install)

Run the binary in a server-side PTY and stream it to
[xterm.js](https://xtermjs.org/) in the browser over a websocket (the
ttyd / gotty / wetty architecture). Users open a URL — no SSH client, no
install at all. More moving parts (a web server, an auth layer, the xterm.js
asset), but the broadest reach. wish also has an experimental web bridge that
can front the same server. Documented here as roadmap.

## Toward a custom terminal client

Standard terminals already speak VT100/ANSI — exactly what bubbletea emits — so
a stock SSH client works out of the box, and you do **not** need to build a
terminal emulator to get started. A *custom* client becomes worthwhile when you
want capabilities a generic terminal can't offer. Keep the server's transport
swappable (the PoC's exec-in-PTY core is independent of SSH) so a custom client
can later add:

- richer input: true mouse reporting, bracketed paste, extra key chords;
- inline graphics (sixel / kitty image protocols);
- file transfer to and from the served app;
- an **app launcher / directory** — pick from the hosted apps after connecting;
- reconnect / session resume across network drops;
- **local-capability bridging** — the big one: let a served app reach the
  *user's* clipboard, files, audio, or printer over a side channel, which
  directly addresses the server-side-I/O caveat below.

The natural evolution is: SSH first (any client), then a custom client that
opens the same session type over the same server and negotiates these extras.

## Caveats — design around these

**Local I/O runs on the SERVER, not the user's machine.** This is the single
most important thing to understand. When an app is served, everything it does
locally happens where it *runs*:

- files it reads and writes (e.g. StarWord opening/saving documents) are on the
  server's disk;
- audio it plays (e.g. the internet-radio demo's `ffmpeg` subprocess) comes out
  of the server;
- clipboard access via `atotto/clipboard`, `SPAWN`ed processes, `os.Getenv`,
  and the working directory are all server-side;
- a single-user state file like `~/.dbtui-state.json` becomes **shared** across
  everyone unless the app keys its state by session identity.

  The PoC demonstrates this concretely: its demo app writes `termserve_proof.txt`
  on startup, and that file appears in the **server's** working directory.

  Design rule: a multi-user served app must derive per-user state from the SSH
  identity (public key / username), not from a process-global home directory.

  One happy exception: bubbletea v1 can copy to the user's real terminal
  clipboard via **OSC 52**, which *does* traverse the terminal, so clipboard-out
  can reach the client even though `atotto/clipboard` cannot.

**Not for Ebitengine apps.** `keen3` and `pacman` are GPU/graphical programs,
not VT100 TUIs. Remote serving here covers terminal apps only.

**Concurrency and sandboxing.** Each session is a running app instance (a child
process in the exec-in-PTY model, plus goroutines). Sessions are sticky — you
can't autoscale them like stateless requests. Before serving untrusted code or
untrusted users, add resource limits and process isolation.

**Auth and multi-tenancy.** Every visitor must prove who they are, and that
identity is what per-user state should be keyed on. `termserve` **requires**
authentication — it refuses to start unless you configure SSH public keys (an
`authorized_keys` allowlist, via `wish.WithAuthorizedKeys`) and/or a password
(`wish.WithPasswordAuth`, compared in constant time). There is no anonymous mode.
Mapping a connection to a stable identity (SSH username / public key) is what lets
a multi-user TermApp keep each person's state separate.

**Offline builds.** `wish`, `charmbracelet/ssh`, and `creack/pty` are not in the
default module cache. The first build fetches them from the network; after that,
`dbasic build --offline` works.

## Try it

See [`examples/termserve/README.md`](../examples/termserve/README.md). In short:

```bash
cd examples/termserve
dbasic build termserve.dbas -o termserve
dbasic build demo/hello_tui.dbas -o demo/hello_tui
cp ~/.ssh/id_ed25519.pub authorized_keys   # allow your key (required)
./termserve demo/hello_tui                    # serves on :2222
# from another terminal:
ssh -p 2222 localhost
```
