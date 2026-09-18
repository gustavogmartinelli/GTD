# Branch workflow

```
feature/* ──PR──> dev ──PR──> master
```

- **`master`** is the release branch. It accepts changes **only** through a pull request **from `dev`**.
- **`dev`** is the integration branch. It accepts changes **only** through a pull request (from any feature branch).
- Nobody pushes directly to `master` or `dev` — not even the repository owner.

## Day-to-day

```bash
git checkout dev
git pull
git checkout -b feature/my-change
# ... work, commit ...
git push -u origin feature/my-change
# open a PR: feature/my-change -> dev
```

When `dev` is ready to release, open a PR `dev -> master`.

## How this is enforced

Two things work together:

1. **Repository rulesets** (`.github/rulesets/`) — block direct pushes, force-pushes and
   deletion on `master` and `dev`, so the only way in is a pull request.
2. **The `source-branch-policy` check** (`.github/workflows/branch-policy.yml`) — runs on every
   PR targeting `master` and fails unless the head branch is `dev` in this repository.
   It is a required status check on `master`, so a PR from any other branch cannot be merged.

GitHub has no native setting for "only accept PRs from branch X", which is why the second
piece is a workflow rather than a checkbox.

## Applying / changing the rules

```bash
gh auth login              # once, as a repo admin
./.github/rulesets/apply.sh
```

The script creates the rulesets, or updates them in place if they already exist. Review the
result at <https://github.com/gustavogmartinelli/GTD/settings/rules>.

Prefer the UI? The equivalent settings are under **Settings → Rules → Rulesets → New branch ruleset**:

| | `master-protection` | `dev-protection` |
|---|---|---|
| Target | `refs/heads/master` | `refs/heads/dev` |
| Restrict deletions | ✅ | ✅ |
| Block force pushes | ✅ | ✅ |
| Require a pull request before merging | ✅ (0 approvals) | ✅ (0 approvals) |
| Require status checks to pass | ✅ `source-branch-policy` | — |

Rulesets apply to everyone by default, including repository admins. If you ever need an
escape hatch, add a bypass actor (Settings → Rules → the ruleset → **Bypass list**) rather
than disabling the ruleset.
