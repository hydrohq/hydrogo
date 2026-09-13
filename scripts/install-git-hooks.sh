#!/bin/sh
set -eu

repo_root="$(git rev-parse --show-toplevel)"
hooks_dir="$repo_root/.git/hooks"
mkdir -p "$hooks_dir"
cp "$repo_root/scripts/git-hooks/pre-commit" "$hooks_dir/pre-commit"
cp "$repo_root/scripts/git-hooks/pre-push" "$hooks_dir/pre-push"
chmod +x "$hooks_dir/pre-commit" "$hooks_dir/pre-push"
echo "HydroGo git hooks installed in $hooks_dir"
