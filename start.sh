#!/usr/bin/env bash
set -euo pipefail

# 一键本地联调：拉起依赖（RabbitMQ via docker compose）+ API + Worker
# 开关（均可环境变量覆盖）：
#   START_RABBITMQ=0   不通过 compose 启动 RabbitMQ
#   START_BACKEND=0    不启动 API
#   START_WORKER=0     不启动 Worker
#   CONFIG_PATH=...    指定后端配置文件（默认拉 compose 依赖时用 config.compose-local.yaml）
#   STOP_DOCKER=1      Ctrl+C 退出时顺便停掉 compose 服务

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$ROOT_DIR"

BACKEND_DIR="${BACKEND_DIR:-$ROOT_DIR/backend}"
RUN_DIR="${RUN_DIR:-$BACKEND_DIR/.run}"

START_REDIS="${START_REDIS:-1}"
START_RABBITMQ="${START_RABBITMQ:-1}"
START_BACKEND="${START_BACKEND:-1}"
START_WORKER="${START_WORKER:-1}"

COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.yml}"
STOP_DOCKER="${STOP_DOCKER:-0}"

# 同时拉 compose 依赖时，默认用指向 3307 端口的 compose-local 配置
if [ -z "${CONFIG_PATH:-}" ] && [ "$START_RABBITMQ" = "1" ]; then
  CONFIG_PATH="configs/config.compose-local.yaml"
fi
export CONFIG_PATH

REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[start.sh] command not found: $1"
    exit 1
  fi
}

BACKEND_PID=""
WORKER_PID=""
cleanup() {
  set +e
  for pid in "${WORKER_PID:-}" "${BACKEND_PID:-}"; do
    if [ -n "$pid" ]; then
      echo "[start.sh] Stopping pid=$pid"
      kill "$pid" >/dev/null 2>&1 || true
      command -v taskkill >/dev/null 2>&1 && taskkill //PID "$pid" //T //F >/dev/null 2>&1 || true
    fi
  done
  if [ "$STOP_DOCKER" = "1" ] && [ -n "${COMPOSE_CMD:-}" ] && [ -f "$COMPOSE_FILE" ]; then
    echo "[start.sh] Stopping docker compose services"
    $COMPOSE_CMD -f "$COMPOSE_FILE" stop >/dev/null 2>&1 || true
  fi
}
trap cleanup INT TERM EXIT

if [ "$START_BACKEND" = "1" ] || [ "$START_WORKER" = "1" ]; then
  [ -d "$BACKEND_DIR" ] || { echo "[start.sh] backend dir not found: $BACKEND_DIR"; exit 1; }
  require_cmd go
  mkdir -p "$RUN_DIR"
fi

detect_compose() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
    return 0
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
    return 0
  fi
  return 1
}

start_rabbitmq_compose() {
  [ -f "$COMPOSE_FILE" ] || { echo "[start.sh] $COMPOSE_FILE not found; skip"; return 0; }
  detect_compose || { echo "[start.sh] docker compose not found; skip"; return 0; }

  echo "[start.sh] Starting RabbitMQ via docker compose ($COMPOSE_FILE)"
  $COMPOSE_CMD -f "$COMPOSE_FILE" up -d rabbitmq

  if command -v docker >/dev/null 2>&1; then
    local rabbit_cid
    rabbit_cid="$($COMPOSE_CMD -f "$COMPOSE_FILE" ps -q rabbitmq 2>/dev/null || true)"
    local i=0
    while [ "$i" -lt 30 ]; do
      if [ -n "$rabbit_cid" ] && docker exec "$rabbit_cid" rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
        echo "[start.sh] RabbitMQ ready"
        return 0
      fi
      sleep 1
      i=$((i + 1))
    done
    echo "[start.sh] RabbitMQ may not be ready yet; continuing anyway"
  fi
}

start_redis() {
  if ! command -v redis-server >/dev/null 2>&1; then
    echo "[start.sh] redis-server not found; skip starting Redis"
    return 0
  fi
  if command -v redis-cli >/dev/null 2>&1 && redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
    echo "[start.sh] Redis already running at $REDIS_HOST:$REDIS_PORT"
    return 0
  fi
  echo "[start.sh] Starting Redis at $REDIS_HOST:$REDIS_PORT"
  nohup redis-server --bind "$REDIS_HOST" --port "$REDIS_PORT" >"$RUN_DIR/redis.log" 2>&1 &
  echo $! >"$RUN_DIR/redis.pid"
}

start_backend_bg() {
  echo "[start.sh] Starting backend (background)"
  (cd "$BACKEND_DIR" && go run ./cmd) &
  BACKEND_PID=$!
  echo "$BACKEND_PID" >"$RUN_DIR/backend.pid"
  echo "[start.sh] Backend PID: $BACKEND_PID"
}

start_worker_bg() {
  echo "[start.sh] Starting worker (background)"
  (cd "$BACKEND_DIR" && go run ./cmd/worker) &
  WORKER_PID=$!
  echo "$WORKER_PID" >"$RUN_DIR/worker.pid"
  echo "[start.sh] Worker PID: $WORKER_PID"
}

if [ "$START_RABBITMQ" = "1" ]; then start_rabbitmq_compose; fi
if [ "$START_REDIS" = "1" ]; then start_redis; fi
if [ "$START_BACKEND" = "1" ]; then start_backend_bg; fi
if [ "$START_WORKER" = "1" ]; then start_worker_bg; fi

if [ "$START_BACKEND" = "1" ] || [ "$START_WORKER" = "1" ]; then
  echo "[start.sh] Press Ctrl+C to stop."
  wait
else
  echo "[start.sh] Nothing to start."
fi
