# Roadmap

Milestone-by-milestone work breakdown. Each milestone ends in a working, demoable state and is **deployed to production** before moving on, so we catch packaging, environment, and TLS issues early instead of all at once on launch day. Within a milestone, items are ordered to be small, mergeable units of work.

## Pre-existing infrastructure

The following is already in place and is **not** repeated in the milestones below:

- Ubuntu VM provisioned, ports 80/443 open
- Caddy installed; Caddyfile reverse-proxies `bandvote.app` to `localhost:8080`
- Cloudflare A record points `bandvote.app` at the VM's external IP
- Let's Encrypt certificate issued by Caddy; HTTPS resolves
- Visiting the site currently returns 503 (no upstream listening yet) — this is the green-light signal that everything in front of the Go binary is correct

## M0 — Repo Bootstrap and Deploy Pipeline

Goal: prove the deploy path works end-to-end before writing any real app code, so every later milestone is just `make deploy`.

- [x] `go mod init`
- [x] Create `cmd/`, `internal/`, `web/templates/`, `web/static/` directories
- [x] `internal/config`: env loading; M0 only needs `PORT` (default `8080`). Later milestones extend it with the Discord and DB vars from spec §11
- [x] `cmd/benefitshow/main.go`: minimal HTTP server bound to `:$PORT` that returns `200 ok`
- [x] Add `Makefile` (`build`, `run`, `lint`, `clean`, `deploy`, `build-linux`)
- [x] Add `.gitignore` (binary, `*.db`, `.env`)
- [x] Commit `README.md`, `spec.md`, `roadmap.md`, `LICENSE` and push to GitHub
- [x] Write `benefitshow.service` (systemd unit) and copy to `/etc/systemd/system/` on the VM
- [x] Create `/opt/benefitshow` with the right ownership for the `make deploy` upload target
- [x] Create `/etc/benefitshow.env` (empty placeholders for now); `chmod 600`
- [x] `systemctl enable --now benefitshow`
- [x] **Deploy:** `make deploy` — VM now serves `200 ok`. `https://bandvote.app` returns `ok` instead of 503.

**Done when:** the VM is serving the placeholder binary over HTTPS at `bandvote.app`, and `make deploy` is a one-command path to ship a new build.

## M1 — Static Web Surface

- [x] CLI subcommand dispatch in `main.go`: `serve`, `import`, `tally` (latter two are stubs)
- [x] `internal/songs`: embedded JSON (`go:embed songs.json`) parsed into a `Song` slice with `ID`, `Title`, `Artist`, `YouTubeURL`, `SpotifyURL`. Start with 5–10 entries; finalize the full list later
- [x] `internal/server`: `net/http` router, basic logging middleware
- [x] `go:embed` for `web/templates/*.html` and `web/static/*`
- [x] `GET /` renders `landing.html`
- [x] `GET /vote` renders `vote.html` with the song list (auth bypassed for now via a hardcoded user)
- [x] `GET /results` renders `results.html` with mock data
- [x] `web/static/style.css` — minimal styling, mobile-first
- [x] `web/static/vote.js` — click-to-rank UI, submit button (POSTs to a not-yet-real endpoint and logs the payload)
- [x] **Deploy:** `make deploy`. Walk the three pages on `bandvote.app` from a phone to verify mobile layout against real TLS and real network conditions.

**Done when:** the three pages render in production and the click-to-rank UI works on desktop and mobile.

## M2 — Persistence

- [x] Add a SQLite driver (`modernc.org/sqlite` or chosen alternative) to `go.mod`
- [x] `internal/ballot`: schema migration on startup, `CreateBallot`, `HasVoted`
- [x] `POST /vote` validates payload (5 distinct song IDs from the pool), inserts a ballot, sets `users.has_voted = true`, all in one transaction
- [x] `GET /vote` for a user with `has_voted = true` renders the locked read-only view of their ranking
- [x] Confirm the SQLite file lands at `DB_PATH` (set in `/etc/benefitshow.env`) and survives a `systemctl restart`
- [x] **Deploy:** `make deploy`. With auth still stubbed, hit `POST /vote` from the prod box (`curl` over SSH) and confirm the row exists and the locked view renders.

**Done when:** prod can accept a ballot from a hardcoded user, persist it, and show the locked view on subsequent loads.

## M3 — Discord Auth

This milestone needs a one-time Discord setup before the deploy step.

- [ ] Register a Discord OAuth app for **local dev** (callback `http://localhost:8080/auth/discord/callback`)
- [ ] Register a separate Discord OAuth app for **prod** (callback `https://bandvote.app/auth/discord/callback`)
- [ ] `internal/auth`: build authorize URL with `state`, exchange code, fetch `/users/@me` and `/users/@me/guilds`
- [ ] Verify `DISCORD_GUILD_ID` is in the user's guild list; reject otherwise
- [ ] Upsert `users` row, mint session token, set HttpOnly Secure SameSite=Lax cookie
- [ ] Auth middleware on `/vote` and `/logout`
- [ ] `POST /logout` clears the session token and redirects to `/`
- [ ] Replace the hardcoded dev user with the real session-derived user
- [ ] Populate `/etc/benefitshow.env` on the VM with prod `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`, `DISCORD_GUILD_ID`, `DISCORD_REDIRECT_URI`, and a fresh `SESSION_SECRET`
- [ ] **Deploy:** `make deploy && systemctl restart benefitshow`. Log in with a real Discord account that's in the guild; confirm `/vote` unlocks. Try with an account that's not in the guild; confirm rejection.

**Done when:** real Discord login on prod authenticates a real guild member and unlocks `/vote`.

## M4 — Tally and Results

- [ ] `benefitshow import <csv>`: parse 5-column rows, insert as `paper` ballots
- [ ] `internal/stv`: implement Droop quota STV. Inputs: ballots and `N`. Output: ordered list of song IDs
- [ ] `benefitshow tally`: run STV across all ballots, print top 10, persist the ordering for the results page
- [ ] `GET /results` reads the persisted ordering and renders the table; empty state if no tally has been run yet
- [ ] Hand-verified test: a small fixture CSV that exercises election, surplus transfer, and elimination
- [ ] **Deploy:** `make deploy`. SSH in, run `./benefitshow import` with the fixture CSV against a scratch DB, then `./benefitshow tally`, and confirm `/results` renders. Reset the DB before opening the polls for real.

**Done when:** on prod, `import` followed by `tally` produces a stable ranked top 10 visible at `/results`.

## Launch and Post-launch

- [ ] Final song list locked in `internal/songs`; ship it via `make deploy`
- [ ] Wipe any test ballots from the prod DB; confirm `users` and `ballots` are empty
- [ ] Open the voting window; share the URL
- [ ] After close: transcribe paper ballots into CSV, `import`, `tally`, share `/results`
- [ ] Wipe the DB or archive the binary + DB after the show
