#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CURRENT_DIR="$SCRIPT_DIR"
REPO_ROOT=""

while true; do
  if [[ -f "$CURRENT_DIR/go.mod" && -f "$CURRENT_DIR/web/package.json" ]]; then
    REPO_ROOT="$CURRENT_DIR"
    break
  fi

  PARENT_DIR="$(dirname "$CURRENT_DIR")"
  if [[ "$PARENT_DIR" == "$CURRENT_DIR" ]]; then
    break
  fi
  CURRENT_DIR="$PARENT_DIR"
done

if [[ -z "$REPO_ROOT" ]]; then
  echo "[ERROR] Could not locate repo root from \"$SCRIPT_DIR\"."
  echo "[ERROR] Expected files: go.mod and web/package.json"
  exit 1
fi

WEB_DIR="$REPO_ROOT/web"

echo "[1/3] Build frontend ..."
(
  cd "$WEB_DIR"
  pnpm build
)

echo "[2/3] Stop processes on ports 9880 and 9881 ..."
KILLED_ANY=0
SEEN_PIDS=";"

kill_pid() {
  local pid="$1"
  local port="$2"

  [[ -z "$pid" ]] && return 0
  case "$SEEN_PIDS" in
    *";$pid;"*) return 0 ;;
  esac

  SEEN_PIDS="${SEEN_PIDS}${pid};"
  echo "[INFO] Killing PID $pid (port $port)..."
  if kill "$pid" >/dev/null 2>&1 || kill -9 "$pid" >/dev/null 2>&1; then
    echo "[OK] Killed PID $pid."
    KILLED_ANY=1
  else
    echo "[WARN] Failed to kill PID $pid (might have exited already)."
  fi
}

kill_by_port() {
  local port="$1"
  local found=0
  local pids=""
  local scanner=""

  if command -v lsof >/dev/null 2>&1; then
    scanner="lsof"
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    scanner="fuser"
    pids="$(fuser -n tcp "$port" 2>/dev/null || true)"
  elif command -v ss >/dev/null 2>&1; then
    scanner="ss"
    pids="$(ss -ltnp "sport = :$port" 2>/dev/null | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' || true)"
  fi

  if [[ -z "$scanner" ]]; then
    echo "[WARN] No port scanner found (need lsof/fuser/ss). Skip cleanup for port $port."
    return 0
  fi

  for pid in $pids; do
    found=1
    kill_pid "$pid" "$port"
  done

  if [[ "$found" -eq 0 ]]; then
    echo "[INFO] Port $port is free."
  fi
}

kill_by_port 9880
kill_by_port 9881

if [[ "$KILLED_ANY" -eq 0 ]]; then
  echo "[INFO] No running process found on ports 9880/9881."
fi

echo "[3/3] Start wails dev ..."
cd "$REPO_ROOT"
exec wails dev "$@"