#!/usr/bin/env bash
set -euo pipefail

# Cloud-init bootstrap script for TerraScore server.
# Expected environment:
#   BRANCH — git branch (set by cloud-init runcmd)
# Expected files written by cloud-init:
#   /root/.ssh/deploy_key — GitHub deploy key
#   /root/.ssh/config     — SSH config for github.com
#   /opt/terrascore/.env   — Application env file

REPO_DIR="/opt/terrascore/repo"
ENV_FILE="/opt/terrascore/.env"
BRANCH="${BRANCH:-main}"

echo "=== TerraScore setup started at $(date -Iseconds) ==="

# Docker should already be installed by cloud-init runcmd.
# Install Docker Compose plugin if not present.
if ! docker compose version &>/dev/null; then
    apt-get install -y docker-compose-plugin
fi

# Ensure Docker is running
systemctl enable docker
systemctl start docker

# Repo should already be cloned by cloud-init runcmd.
# If not, clone it now.
if [ ! -d "$REPO_DIR/.git" ]; then
    git clone -b "$BRANCH" git@github.com:$(cd "$REPO_DIR" 2>/dev/null && git remote get-url origin | sed 's|.*github.com[:/]||;s|\.git$||' || echo "OWNER/REPO").git "$REPO_DIR" || true
fi

cd "$REPO_DIR"

# Symlink .env into repo root for docker-compose env_file
ln -sf "$ENV_FILE" "$REPO_DIR/.env"

# Save branch for update script
echo "$BRANCH" > /opt/terrascore/branch

# Export version for docker compose build args
export VERSION=$(cat VERSION 2>/dev/null || echo "dev")

# Source env vars for DB connection
set -a
source "$ENV_FILE"
set +a

# Build and start all services
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build --force-recreate

# Wait for API to be healthy before running migrations
echo "Waiting for API to be ready..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo "API is healthy."
        break
    fi
    echo "  attempt $i/30..."
    sleep 5
done

# Run migrations
ENCODED_PW=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${DB_PASSWORD:-terrascore}', safe=''))")
DB_URL="postgres://${DB_USER:-terrascore}:${ENCODED_PW}@terrascore-postgres:5432/${DB_NAME:-terrascore}?sslmode=disable"
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T api /bin/terrascore-migrate -direction up -path /app/db/migrations -db "$DB_URL" || \
    echo "Migration failed or already applied — skipping."

# Install cron for auto-update
CRON_LINE="*/10 * * * * BRANCH=$BRANCH /opt/terrascore/repo/infra/deploy/update.sh >> /var/log/terrascore-update.log 2>&1"
(crontab -l 2>/dev/null | grep -v "update.sh"; echo "$CRON_LINE") | crontab -

echo "=== TerraScore setup completed at $(date -Iseconds) ==="
