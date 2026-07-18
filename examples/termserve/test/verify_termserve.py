#!/usr/bin/env python3
# =========================================================================
# verify_termserve.py — end-to-end check for the termserve PoC.
#
# Connects the real `ssh` client (in a pty) to a running termserve, renders
# the result through a pyte VT100 emulator, and asserts that:
#   1. the served TUI paints over SSH,
#   2. the initial client window size reaches the app, and
#   3. a live resize propagates through ssh -> termserve -> the app's PTY.
#
# termserve must already be listening (see run_verify.sh, which starts it in
# an isolated temp dir first). Requires: ssh client, python3, pyte.
# Exit 0 = all assertions pass; 1 = a failure; 2 = launch error.
# =========================================================================
import sys, os, pty, time, select, fcntl, termios, struct
try:
    import pyte
except ImportError:
    sys.stderr.write("verify_termserve: pyte not installed (pip install pyte)\n")
    sys.exit(2)

HOST = os.environ.get("HOST", "localhost")
PORT = os.environ.get("PORT", "2222")
COLS0, ROWS0 = 90, 24
COLS1, ROWS1 = 100, 30

def set_winsize(fd, cols, rows):
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))

def pump(fd, stream, seconds):
    deadline = time.time() + seconds
    while time.time() < deadline:
        r, _, _ = select.select([fd], [], [], 0.2)
        if fd in r:
            try:
                data = os.read(fd, 65536)
            except OSError:
                return False
            if not data:
                return False
            stream.feed(data)
    return True

def text(screen):
    return "\n".join(screen.display)

def wait_for(fd, stream, screen, want, timeout=15):
    deadline = time.time() + timeout
    while time.time() < deadline:
        pump(fd, stream, 0.3)
        if want in text(screen):
            return True
    return False

def main():
    screen = pyte.Screen(COLS0, ROWS0)
    stream = pyte.ByteStream(screen)
    # termserve always requires authentication. Log in either with an SSH key
    # (SSH_KEY, whose public half run_verify.sh put in authorized_keys) or a
    # password (SSH_PASSWORD, fed via sshpass). Exactly one is expected.
    common = ["-tt", "-p", PORT,
              "-o", "StrictHostKeyChecking=no",
              "-o", "UserKnownHostsFile=/dev/null",
              "-o", "GlobalKnownHostsFile=/dev/null",
              "-o", "LogLevel=ERROR"]
    key = os.environ.get("SSH_KEY")
    password = os.environ.get("SSH_PASSWORD")
    if password:
        os.environ["SSHPASS"] = password
        argv = ["sshpass", "-e", "ssh"] + common + [
            "-o", "PreferredAuthentications=password",
            "-o", "PubkeyAuthentication=no", HOST]
    else:
        argv = ["ssh"] + common + [
            "-o", "IdentitiesOnly=yes",
            "-o", "PreferredAuthentications=publickey",
            "-i", key, HOST]
    prog = argv[0]
    pid, fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        set_winsize(0, COLS0, ROWS0)
        os.execvp(prog, argv)
        os._exit(127)

    set_winsize(fd, COLS0, ROWS0)
    ok = True

    if not wait_for(fd, stream, screen, "TERMSERVE DEMO"):
        print("FAIL: banner 'TERMSERVE DEMO' never rendered"); print(text(screen)); ok = False
    else:
        print("PASS: served TUI rendered over SSH")

    if ok and not wait_for(fd, stream, screen, f"{COLS0} x {ROWS0}"):
        print(f"FAIL: initial size '{COLS0} x {ROWS0}' not shown"); print(text(screen)); ok = False
    elif ok:
        print(f"PASS: initial window size {COLS0} x {ROWS0} propagated")

    if ok:
        screen.resize(ROWS1, COLS1)
        set_winsize(fd, COLS1, ROWS1)
        if not wait_for(fd, stream, screen, f"{COLS1} x {ROWS1}"):
            print(f"FAIL: resized size '{COLS1} x {ROWS1}' not shown"); print(text(screen)); ok = False
        else:
            print(f"PASS: live resize {COLS1} x {ROWS1} propagated to the app")

    try:
        os.write(fd, b"q"); time.sleep(0.5)
    except OSError:
        pass
    try:
        os.close(fd)
    except OSError:
        pass
    try:
        os.waitpid(pid, 0)
    except OSError:
        pass
    sys.exit(0 if ok else 1)

if __name__ == "__main__":
    main()
