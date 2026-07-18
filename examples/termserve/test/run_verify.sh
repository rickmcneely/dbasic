#!/usr/bin/env bash
# =========================================================================
# run_verify.sh — build termserve + the demo TermApp, then prove the whole
# thing end-to-end in an isolated temp dir:
#   1. termserve REFUSES to start with no authentication configured.
#   2. With SSH-key auth, a real ssh session renders the TermApp and a live
#      resize propagates (via verify_termserve.py + pyte).
#   3. Password auth works (checked with sshpass, if available).
#   4. A key that is NOT in authorized_keys is rejected.
#
# Everything (host key, authorized_keys, throwaway keypairs, the server-side
# proof file) lives under a scratch dir, never in the repo.
# Requires: dbasic on PATH, ssh, ssh-keygen, python3 + pyte. sshpass optional.
# Usage:  ./run_verify.sh
# =========================================================================
set -uo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
TERMSERVE_DIR="$(dirname "$HERE")"
PORT="${PORT:-2222}"
PASSWORD="verify-pw-$$"
DBASIC="${DBASIC:-dbasic}"

echo "== building termserve and the demo TermApp =="
( cd "$TERMSERVE_DIR" && "$DBASIC" build termserve.dbas -o termserve ) || exit 1
( cd "$TERMSERVE_DIR" && "$DBASIC" build demo/hello_tui.dbas -o demo/hello_tui ) || exit 1

RUN="$(mktemp -d)"
cleanup() { [[ -n "${SRV:-}" ]] && kill "$SRV" 2>/dev/null; rm -rf "$RUN"; }
trap cleanup EXIT

APP="$TERMSERVE_DIR/demo/hello_tui"
SERVER="$TERMSERVE_DIR/termserve"
fail() { echo "FAIL: $*"; exit 1; }

# --- 1. No auth configured -> the server must refuse to start ---------------
echo "== TEST 1: refuse to start with no authentication =="
NOAUTH="$(mktemp -d)"
( cd "$NOAUTH" && env -u TERMSERVE_PASSWORD TERMSERVE_ADDR=":$PORT" "$SERVER" "$APP" ) \
    > "$NOAUTH/out.log" 2>&1
rc=$?
rm -rf "$NOAUTH"
[[ $rc -ne 0 ]] || fail "server started with no auth (should have refused)"
echo "PASS: refused to start with no auth (exit $rc)"

# --- Make a throwaway keypair (authorized) and a second one (not authorized) -
ssh-keygen -q -t ed25519 -N "" -f "$RUN/id_ok"  || fail "ssh-keygen (ok) failed"
ssh-keygen -q -t ed25519 -N "" -f "$RUN/id_bad" || fail "ssh-keygen (bad) failed"
cp "$RUN/id_ok.pub" "$RUN/authorized_keys"

# --- Start the real server with BOTH key auth and password auth -------------
echo "== starting termserve on :$PORT (isolated dir: $RUN) =="
( cd "$RUN" && TERMSERVE_ADDR=":$PORT" TERMSERVE_PASSWORD="$PASSWORD" "$SERVER" "$APP" \
    > "$RUN/termserve.log" 2>&1 ) &
SRV=$!
sleep 1.5
kill -0 "$SRV" 2>/dev/null || { echo "server failed to start:"; cat "$RUN/termserve.log"; exit 1; }

# --- 2. Key auth: full render + live-resize test via pyte -------------------
echo "== TEST 2: SSH-key login renders the TermApp + live resize =="
PORT="$PORT" SSH_KEY="$RUN/id_ok" python3 "$HERE/verify_termserve.py"
RC=$?
[[ $RC -eq 0 ]] || fail "key-auth render/resize test failed"

# --- 3. Password auth (best-effort; needs sshpass) --------------------------
if command -v sshpass >/dev/null 2>&1; then
    echo "== TEST 3: password login renders the TermApp =="
    PORT="$PORT" SSH_PASSWORD="$PASSWORD" python3 "$HERE/verify_termserve.py" \
        || fail "password-auth render test failed"
else
    echo "== TEST 3: skipped (sshpass not installed) =="
fi

# --- 4. An unauthorized key must be rejected --------------------------------
echo "== TEST 4: a key not in authorized_keys is rejected =="
if ssh -p "$PORT" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
       -o GlobalKnownHostsFile=/dev/null -o LogLevel=ERROR -o BatchMode=yes \
       -o IdentitiesOnly=yes -o PreferredAuthentications=publickey \
       -i "$RUN/id_bad" localhost true 2>/dev/null; then
    fail "unauthorized key was accepted"
fi
echo "PASS: unauthorized key rejected"

echo "== server log =="
cat "$RUN/termserve.log"
echo "== server-side proof file (written by the TermApp, ON THE SERVER) =="
ls -l "$RUN/termserve_proof.txt" && cat "$RUN/termserve_proof.txt"

echo "ALL TESTS PASSED"
exit 0
