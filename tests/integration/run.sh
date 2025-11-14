#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
COMMAND="${1:-up}"

wait_for_port() {
  local host="$1"
  local port="$2"
  local timeout="${3:-60}"
  local elapsed=0

  echo "Waiting for ${host}:${port} to become available..."
  while ! nc -z "${host}" "${port}" >/dev/null 2>&1; do
    sleep 2
    elapsed=$((elapsed + 2))
    if [ "${elapsed}" -ge "${timeout}" ]; then
      echo "Timed out waiting for ${host}:${port}"
      exit 1
    fi
  done
  echo "${host}:${port} is ready."
}

case "${COMMAND}" in
  up)
    docker compose -f "${COMPOSE_FILE}" up -d
    wait_for_port "localhost" 55432 90
    wait_for_port "localhost" 54222 60
    ;;
  down)
    docker compose -f "${COMPOSE_FILE}" down -v
    ;;
  *)
    echo "Usage: $0 {up|down}"
    exit 1
    ;;
esac

