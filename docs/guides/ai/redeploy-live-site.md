# Redeploying code changes to an already-live site (AI guide)

Audience: an agent that needs to ship a code change to a site that's
**already deployed** (not a first-time setup — see
`deploy-ftp-shared-hosting.md` or `deploy-vps.md` for that). This is the
loop used repeatedly once a site is live: merge → build → upload → push.

## The loop

1. **Confirm the working tree is clean and on the right branch.**
```sh
git status --short
git log --oneline -3
```
If the change under review is on a feature branch, merge it into `main`
first (fast-forward preferred, no unrelated changes mixed in):
```sh
git checkout main
git pull origin main
git merge <feature-branch> --no-edit
```

2. **Rebuild the binary** matching the target's OS/arch (shared hosting
and most VPS: `linux/amd64`, pure Go, no C toolchain needed either side):
```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o dist/hydro ./cmd/hydro
```

3. **Check whether anything besides the binary changed** that also needs
   deploying — a new/changed `.htaccess`, a new static asset, a theme
   file. Diff against what was last deployed if unsure:
```sh
git diff <last-deployed-commit> HEAD --stat
```
   Don't assume "I only changed Go code" without checking — a change to
   `deploy/hosting/.htaccess` or a theme file needs its own upload.

4. **Upload.**
   - FTP shared hosting: same `curl -T` pattern as the initial deploy guide,
     targeting `private/hydro` (or wherever the binary lives). Verify the
     uploaded byte count matches the local file.
   - VPS: `scp` the binary over, then restart the service —
     `ssh <user>@<host> "sudo systemctl restart hydrogo"`. A VPS deploy
     needs this restart step; shared/CGI hosting does not (every request
     already spawns a fresh process, so the next request after upload
     picks up the new binary automatically).

5. **Run any new migrations** if the change included one — same `migrate`
   command as initial setup, safe to re-run (idempotent, already-applied
   migrations are skipped). See `database-migrations.md` for adding new
   ones correctly.

6. **Push to the remote git repo** (keep history in sync with what's
   actually live):
```sh
git push origin main
```
   If this repo's pre-push hook includes test/lint gates you want to skip
   for a docs-only or already-verified change, check whether it supports a
   tag like `#notest` in the commit message (this repo's does — see its
   hook script) rather than bypassing the hook entirely with `--no-verify`.

7. **Verify live.** At minimum:
```sh
curl -s -o /dev/null -w '%{http_code}\n' "https://<domain>/"
```
   For anything touching auth, caching, or a specific route, hit that
   route specifically and check response headers/status, not just the
   homepage. If the change affects something cached (see
   `docs/architecture/0265-fastcgi-and-persistent-html-cache.md`), remember
   a stale cached copy may still be served until it's purged or the
   relevant content changes — don't mistake a cache hit for the deploy not
   having worked.

## Push race condition

If two deploy pushes happen close together (e.g. you pushed, then
immediately pushed again after a quick follow-up commit), you can hit:
```
! [remote rejected]   main -> main (cannot lock ref 'refs/heads/main': is at <X> but expected <Y>)
```
This is a benign ref-lock race, not a real conflict — your local `main` is
still correct, just re-run `git push origin main`.

## What NOT to do

- Don't skip the "did anything besides Go code change" check (step 3) —
  the most common silent-failure mode in this workflow is updating the
  binary but forgetting a config/static-asset change that shipped in the
  same commit.
- Don't forget the VPS restart step — unlike CGI hosting, an already-running
  `hydro serve` process keeps serving the old binary from memory until
  explicitly restarted; uploading a new file alone does nothing until then.
- Don't push to the remote repo before confirming the live site actually
  works — if something's broken, you want to know and fix it before the
  git history says it's done.
