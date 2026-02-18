#!/usr/bin/env bash
set -euo pipefail

# Cloud-init bootstrap script for LandIntel server.
# Expected environment:
#   BRANCH — git branch (set by cloud-init runcmd)
# Expected files written by cloud-init:
#   /root/.ssh/deploy_key — GitHub deploy key
#   /root/.ssh/config     — SSH config for github.com
#   /opt/landintel/.env   — Application env file

REPO_DIR="/opt/landintel/repo"
ENV_FILE="/opt/landintel/.env"
BRANCH="${BRANCH:-main}"

echo "=== LandIntel setup started at $(date -Iseconds) ==="

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
echo "$BRANCH" > /opt/landintel/branch

# Build and start all services
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# Wait for API to be ready, then run migrations
echo "Waiting for postgres to be healthy..."
sleep 10
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T api /bin/landintel-api migrate up 2>/dev/null || \
    echo "Migration command not available or already applied — skipping."

# Install cron for auto-update
CRON_LINE="*/10 * * * * BRANCH=$BRANCH /opt/landintel/repo/infra/deploy/update.sh >> /var/log/landintel-update.log 2>&1"
(crontab -l 2>/dev/null | grep -v "update.sh"; echo "$CRON_LINE") | crontab -

echo "=== LandIntel setup completed at $(date -Iseconds) ==="
