#!/usr/bin/env python3
# =========================================================================
# vtdrive.py — drive a built TUI program under a pty, feed it keystrokes,
# render the result through a pyte VT100 emulator, and assert that an
# expected substring is (or is NOT) present in the final screen.
#
# WHY pyte (not a naive ANSI strip): bubbletea emits cursor-positioning
# diffs, not a clean linear stream — only a real terminal emulator
# reconstructs the visible screen correctly.
#
# Used by smoke_widgets.sh. Standalone usage:
#   python3 vtdrive.py <prog> <keys> [--want STR | --not STR] ...
#     <keys>  comma-separated; tokens: tab,up,down,left,right,enter,space,
#             esc, hold:<seconds>, or a literal single char.
#   --want STR   assertion: STR must appear in the final screen
#   --not  STR   assertion: STR must NOT appear
# Exit 0 = all assertions pass; 1 = a failure; 2 = launch error.
# With no assertions it just prints the screen (calibration mode).
# =========================================================================
import sys, os, pty, time, select, fcntl, termios, struct
try:
    import pyte
except ImportError:
    sys.stderr.write("vtdrive: pyte not installed (pip install pyte)\n")
    sys.exit(2)

COLS, ROWS = 100, 40
KEYMAP = {"tab": "\t", "up": "\x1b[A", "down": "\x1b[B", "left": "\x1b[D",
          "right": "\x1b[C", "enter": "\r", "space": " ", "esc": "\x1b"}


def run(prog, keyspec):
    screen = pyte.Screen(COLS, ROWS)
    stream = pyte.ByteStream(screen)
    pid, fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        os.execvp(prog, [prog])
        os._exit(127)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))

    def pump(duration):
        end = time.time() + duration
        while time.time() < end:
            r, _, _ = select.select([fd], [], [], 0.1)
            if r:
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    return
                if data:
                    stream.feed(data)

    time.sleep(0.8)
    pump(1.0)
    for tok in keyspec.split(","):
        if not tok:
            continue
        if tok.startswith("hold:"):
            pump(float(tok[5:]))
            continue
        os.write(fd, KEYMAP.get(tok, tok).encode())
        pump(0.35)
    text = "\n".join(line.rstrip() for line in screen.display)
    try:
        os.write(fd, b"\x03")
        time.sleep(0.2)
        os.kill(pid, 9)
    except Exception:
        pass
    return text


def main():
    if len(sys.argv) < 3:
        sys.stderr.write("usage: vtdrive.py <prog> <keys> [--want STR|--not STR]...\n")
        sys.exit(2)
    prog, keyspec = sys.argv[1], sys.argv[2]
    # Parse assertions: list of (kind, string) where kind in {"want","not"}.
    asserts = []
    i = 3
    while i < len(sys.argv):
        if sys.argv[i] in ("--want", "--not") and i + 1 < len(sys.argv):
            asserts.append((sys.argv[i][2:], sys.argv[i + 1]))
            i += 2
        else:
            i += 1

    screen = run(prog, keyspec)

    if not asserts:
        print(screen)
        return

    failed = False
    for kind, needle in asserts:
        present = needle in screen
        ok = present if kind == "want" else not present
        tag = "ok  " if ok else "FAIL"
        sign = "want" if kind == "want" else "not "
        print(f"  [{tag}] {sign}: {needle!r}")
        if not ok:
            failed = True
    if failed:
        print("--- final screen ---")
        for ln in screen.split("\n"):
            if ln.strip():
                print("   " + ln[:90])
        sys.exit(1)


if __name__ == "__main__":
    main()
