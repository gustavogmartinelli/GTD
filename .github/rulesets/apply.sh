#!/usr/bin/env bash
# Applies the branch rulesets in this directory to the repository.
#
# Requires the GitHub CLI, authenticated as a user with admin rights on the repo:
#   gh auth login
#   ./.github/rulesets/apply.sh
#
# Re-running is safe: a ruleset with the same name is updated in place.

set -euo pipefail

REPO="${REPO:-gustavogmartinelli/GTD}"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

command -v gh >/dev/null || { echo "gh (GitHub CLI) is required: https://cli.github.com" >&2; exit 1; }

existing="$(gh api "repos/$REPO/rulesets" --jq '.[] | "\(.name)\t\(.id)"')"

apply() {
  local name="$1" file="$2" id
  id="$(printf '%s\n' "$existing" | awk -F'\t' -v n="$name" '$1 == n { print $2; exit }')"

  if [ -n "$id" ]; then
    echo "Updating ruleset '$name' (id $id) on $REPO"
    gh api --method PUT "repos/$REPO/rulesets/$id" --input "$file" --jq '.name + " -> " + .enforcement'
  else
    echo "Creating ruleset '$name' on $REPO"
    gh api --method POST "repos/$REPO/rulesets" --input "$file" --jq '.name + " -> " + .enforcement'
  fi
}

apply "dev-protection"    "$DIR/dev.json"
apply "master-protection" "$DIR/master.json"

echo
echo "Done. Review them at: https://github.com/$REPO/settings/rules"
