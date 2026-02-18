#!/usr/bin/env bash
set -euo pipefail

# Auto-pull and rebuild script.
# Called by cron every 10 minutes and by CI SSH on demand.
# Set FORCE=1 to rebuild even if no changes detected.

REPO_DIR="/opt/terrascore/repo"
BRANCH="${BRANCH:-$(cat /opt/terrascore/branch 2>/dev/null || echo main)}"
FORCE="${FORCE:-0}"

cd "$REPO_DIR"

# Fetch latest
git fetch origin "$BRANCH"

LOCAL_SHA=$(git rev-parse HEAD)
REMOTE_SHA=$(git rev-parse "origin/$BRANCH")

if [ "$LOCAL_SHA" = "$REMOTE_SHA" ] && [ "$FORCE" != "1" ]; then
    exit 0
fi

echo "[$(date -Iseconds)] Updating: $LOCAL_SHA -> $REMOTE_SHA (force=$FORCE)"

# Pull latest code
git reset --hard "origin/$BRANCH"

# Ensure .env symlink
ln -sf /opt/terrascore/.env "$REPO_DIR/.env"

# Rebuild and restart only changed services
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

echo "[$(date -Iseconds)] Update complete: $(git rev-parse --short HEAD)"
