#!/usr/bin/env bash
set -euo pipefail

# ── colours ─────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[dev]${NC} $*"; }
warn()  { echo -e "${YELLOW}[dev]${NC} $*"; }
error() { echo -e "${RED}[dev]${NC} $*"; exit 1; }

# ── deps check ───────────────────────────────────────────────────────────────
command -v docker >/dev/null || error "docker is required"
command -v go     >/dev/null || error "go is required"
command -v npm    >/dev/null || error "npm is required"

# ── config ───────────────────────────────────────────────────────────────────
DB_CONTAINER="recipeasy-pg"
DB_USER="recipeasy"
DB_PASS="localpass"
DB_NAME="recipeasy"
DB_PORT="5432"
API_PORT="8080"
IMAGE_DIR="/tmp/recipeasy-images"

export DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"
export JWT_SECRET="dev-secret-not-for-production"
export GOOGLE_CLIENT_ID=""
export GOOGLE_CLIENT_SECRET=""
export GOOGLE_REDIRECT_URL="http://localhost:${API_PORT}/v1/auth/google/callback"
export ALLOWED_EMAILS="dev@local"
export GEMINI_API_KEY="${GEMINI_API_KEY:-}"
export FRONTEND_URL="http://localhost:5173"
export IMAGE_DIR
export DEV_AUTH="true"
export PORT="${API_PORT}"

mkdir -p "${IMAGE_DIR}"

# ── cleanup on exit ──────────────────────────────────────────────────────────
cleanup() {
  info "shutting down..."
  kill "${API_PID:-}" "${WEB_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# ── postgres ─────────────────────────────────────────────────────────────────
if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
  info "postgres already running"
elif docker ps -a --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
  info "starting existing postgres container"
  docker start "${DB_CONTAINER}" >/dev/null
else
  info "creating postgres container"
  docker run -d \
    --name "${DB_CONTAINER}" \
    -e POSTGRES_USER="${DB_USER}" \
    -e POSTGRES_PASSWORD="${DB_PASS}" \
    -e POSTGRES_DB="${DB_NAME}" \
    -p "${DB_PORT}:5432" \
    postgres:16-alpine >/dev/null
fi

info "waiting for postgres..."
until docker exec "${DB_CONTAINER}" pg_isready -U "${DB_USER}" -q 2>/dev/null; do
  sleep 0.5
done

MIGRATION_FILE="$(cd "$(dirname "$0")" && pwd)/api/db/migrations/001_initial_schema.sql"
info "applying schema migration"
# Strip the goose Down section — psql would run it and drop the tables we just created
sed '/^-- +goose Down/,$d' "${MIGRATION_FILE}" | \
  docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1

# ── go api ───────────────────────────────────────────────────────────────────
info "starting Go API on :${API_PORT} (DEV_AUTH=true)"
(cd "$(dirname "$0")/api" && go run ./cmd/server) &
API_PID=$!

# Wait for API to be ready
for i in $(seq 1 20); do
  if curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done
info "API ready"

# ── vite dev server ──────────────────────────────────────────────────────────
info "starting Vite dev server on :5173"
(cd "$(dirname "$0")/web" && npm run dev) &
WEB_PID=$!

warn ""
warn "  App:        http://localhost:5173"
warn "  API:        http://localhost:${API_PORT}"
warn "  Dev login:  http://localhost:5173/login  → click 'Dev login'"
warn ""
warn "  Press Ctrl+C to stop"
warn ""

wait
