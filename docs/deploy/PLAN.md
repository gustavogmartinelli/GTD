# Deploy plan: GTD on Oracle Cloud Always Free

Decisions for the first deploy cycle (agreed 2026-10-04). The step-by-step
setup is in [oracle-setup.md](oracle-setup.md).

## Decisions

| Topic | Decision | Why |
|---|---|---|
| Audience | Private: only the owner | No auth yet; anyone with the URL could read or delete items |
| Host | Oracle Always Free **Ampere A1** (arm64), São Paulo, Ubuntu 24.04 | Performance; free. Start 1 OCPU / 6 GB, resize to 2 / 12 |
| Account | Free tier only, no Pay As You Go | Zero cost risk; revisit if the VM gets reclaimed |
| Environments | One VM (`gtd-test`) | Enough for now; the setup is scripted so cloning it is cheap |
| Runtime | Docker + compose, `distroless/static` image | Isolated, reproducible; ~15 MB image |
| Registry | GHCR, private, tagged by git SHA (+ moving branch tag) | Fits GitHub Actions; SHA tags make rollback a one-line change |
| Access | Tailscale; app bound to `127.0.0.1`, published with `tailscale serve` (HTTPS) | No public app port; works from the phone on mobile data; HTTPS for the future web UI |
| Deploy | GitHub Actions: push to `dev` → test → build arm64 → push → SSH over Tailscale → health-gated `up -d` with auto-rollback | Every deploy traceable to a commit; nothing depends on the PC |
| Branches | PRs: vet + test. `dev`: deploy to test VM. `master`: build only | `master` deploys to prod once a prod VM exists |
| Database | SQLite now, Postgres later | Get the pipeline running first; the migration is small |
| Migrations | goose, embedded, applied at startup | Safe with a single instance; supports SQLite and Postgres |
| Backups | Nightly `sqlite3 .backup` → Object Storage via rclone; 30 dailies + monthly forever; healthchecks.io alert on failure | Data is never purged automatically; silent backup failure is the main risk |
| Idle reclamation | Accept the risk: the VM is disposable (cloud-init + backups); upgrade only if it happens | Free-tier-only accounts can lose idle instances |
| Monitoring | `/healthz`, Docker HEALTHCHECK, `restart: unless-stopped`, deploy fails if unhealthy for 30 s | Enough for a single-user test environment |

## Files

| Path | Role |
|---|---|
| `Dockerfile` | Cross-compiling multi-stage build → `distroless/static:nonroot` |
| `deploy/compose.yaml` | Service definition on the VM (`/opt/gtd`) |
| `deploy/cloud-init.yaml` | First-boot setup: Docker, Tailscale, swap, deploy user, backup cron |
| `deploy/scripts/oci-create-instance.sh` | Retries instance creation until A1 capacity appears |
| `deploy/scripts/remote-deploy.sh` | Health-gated deploy with rollback (run by CI on the VM) |
| `deploy/scripts/backup.sh` / `restore.sh` | Nightly backup and restore |
| `.github/workflows/ci.yml` | PR checks |
| `.github/workflows/deploy.yml` | Build and deploy |

## Next cycles (roadmap)

1. **Auth** (single user first), so the app could be opened beyond the tailnet
2. **Web UI**, mobile-friendly (served over the existing `tailscale serve` HTTPS)
3. **Postgres**: new `adapter/repository/postgres` behind `usecase.ItemRepository`,
   a goose migration set for Postgres, and a one-off SQLite → Postgres data copy
4. **Multiple users**
5. **Separate test VM**, with `master` deploying to the current VM as prod
   (the free tier has room for a second A1 VM with 2 OCPU / 12 GB)
