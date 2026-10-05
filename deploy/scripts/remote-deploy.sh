#!/usr/bin/env bash
# Runs ON THE VM, invoked by the GitHub Actions deploy job:
#   /opt/gtd/remote-deploy.sh <image-tag>
# Pins IMAGE_TAG in .env, pulls, restarts, and waits for /healthz. If the
# new version never gets healthy it rolls back to the previous tag and
# exits non-zero so the workflow run fails.
set -euo pipefail

TAG="${1:?usage: remote-deploy.sh <image-tag>}"
cd /opt/gtd

PREV="$(sed -n 's/^IMAGE_TAG=//p' .env 2>/dev/null || true)"

set_tag() {
  if grep -q '^IMAGE_TAG=' .env 2>/dev/null; then
    sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$1/" .env
  else
    echo "IMAGE_TAG=$1" >> .env
  fi
}

healthy() {
  for _ in $(seq 1 30); do
    if curl -fsS -m 2 http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

echo "deploying $TAG (previous: ${PREV:-none})"
set_tag "$TAG"
docker compose pull
docker compose up -d

if healthy; then
  echo "healthy on $TAG"
  docker image prune -f >/dev/null
  exit 0
fi

echo "!! $TAG did not become healthy within 30s; recent logs:"
docker compose logs --tail 50 gtd || true

if [ -n "$PREV" ] && [ "$PREV" != "$TAG" ]; then
  echo "rolling back to $PREV"
  set_tag "$PREV"
  docker compose up -d
  healthy && echo "rollback to $PREV is healthy" || echo "!! rollback is ALSO unhealthy"
fi
exit 1
