#!/usr/bin/env bash
set -euo pipefail

ZPLAY="go run ./cmd/zplay"
TEST_NAME="inttest-$(date +%s | tail -c 6)"
PASS=0
FAIL=0

log() { echo "=== $1 ==="; }
pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

log "Deploy vanilla Terraria server: $TEST_NAME"
if $ZPLAY deploy --game terraria --variant vanilla --name "$TEST_NAME" --memory 4Gi --node oracle1 --port 7778 --auto-backup 2>&1; then
  pass "deploy"
else
  fail "deploy"
  echo "Cannot continue without deploy. Exiting."
  exit 1
fi

log "Status"
if $ZPLAY status "$TEST_NAME" 2>&1 | grep -q "Server:"; then
  pass "status"
else
  fail "status"
fi

log "List (text)"
if $ZPLAY list 2>&1 | grep -q "$TEST_NAME"; then
  pass "list text"
else
  fail "list text"
fi

log "List (JSON)"
if $ZPLAY list --json 2>&1 | grep -q "\"name\": \"$TEST_NAME\""; then
  pass "list json"
else
  fail "list json"
fi

log "Backup"
if $ZPLAY backup "$TEST_NAME" 2>&1; then
  pass "backup"
else
  fail "backup"
fi

log "Stop"
if $ZPLAY stop "$TEST_NAME" 2>&1 | grep -q "stopped"; then
  pass "stop"
else
  fail "stop"
fi

log "Start"
if $ZPLAY start "$TEST_NAME" 2>&1 | grep -q "started"; then
  pass "start"
else
  fail "start"
fi

log "Delete"
if $ZPLAY delete "$TEST_NAME" --yes 2>&1 | grep -q "deleted"; then
  pass "delete"
else
  fail "delete"
fi

log "Cleanup"
$ZPLAY cleanup --yes 2>&1 || true
pass "cleanup (ran)"

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
