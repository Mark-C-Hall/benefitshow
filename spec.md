# benefitshow — Specification

## 1. Overview

benefitshow is a ranked-choice voting tool for selecting a small setlist (5 songs played, with a ranked top 10 produced for flexibility) from a fixed pool of ~30 candidate songs. The voting population is a single online community (Discord guild) of roughly 50–100 people, with a hybrid online + paper voting workflow.

The app is intentionally narrow: one election, one fixed song pool, one administrator. There is no admin UI, no multi-tenancy, and no support for multiple concurrent elections.

## 2. Goals and Non-goals

### Goals

- Authenticated online voting via Discord OAuth, restricted to members of a configured guild.
- Click-to-rank UI that works on phones and desktop without a JavaScript framework.
- Ability to merge in paper ballots transcribed by the administrator after the online window closes.
- Multi-winner ranked-choice tally producing a ranked top 10.
- Single-binary deploy with no runtime dependencies beyond a SQLite file.

### Non-goals

- No admin UI for managing songs, voters, or election windows. All such state is hardcoded or set via environment variables.
- No password-based or email-based auth.
- No support for editing or revoking a submitted ballot. One submission per user, locked on submit.
- No live results or real-time updates. Results are produced by running an offline tally command.
- No automated test suite. The codebase is small enough to verify by hand within the build window.

## 3. Voting Workflow

1. The administrator publishes a song list (paper handouts and an online page) with stable IDs (`01`–`30+`). The same IDs appear on paper ballots and in the app.
2. Voters either:
   - Log in to the app with Discord, rank 5 songs, and submit; or
   - Fill out a paper ballot in person.
3. The online voting window runs ~10 days, spanning two weekly gatherings to maximize paper-ballot coverage.
4. At close, the administrator transcribes paper ballots into a CSV (one row per ballot, 5 columns of song IDs in rank order) and imports them via `benefitshow import`.
5. The administrator runs `benefitshow tally`. The STV algorithm produces a ranked top 10, written to the `/results` page and printed to stdout.
6. The top picks are passed to the band offline. Any songs that cannot be performed are skipped and the next-ranked song moves up. This veto/backfill step happens outside the app.

## 4. Architecture

A single Go binary serves the web app, embeds its templates and static assets via `go:embed`, and persists state to a SQLite file in its working directory. The same binary handles the `serve`, `import`, and `tally` subcommands.

```
┌──────────────┐  HTTPS  ┌─────────┐  :8080  ┌──────────────┐
│   browser    │ ──────► │  Caddy  │ ──────► │  benefitshow │
└──────────────┘         └─────────┘         │   (Go bin)   │
                                             └──────┬───────┘
                                                    │
                                              ┌─────▼──────┐
                                              │ SQLite DB  │
                                              └────────────┘
```

Caddy terminates TLS and reverse-proxies to the Go binary. The binary serves all routes, including its own static assets (no separate static host).

## 5. Data Model

Songs are not stored in the database; they live in a hardcoded Go slice in `internal/songs`. Their IDs match the paper-ballot IDs (`01`–`30+`).

### `users`

| Column          | Type      | Notes                                |
| --------------- | --------- | ------------------------------------ |
| `id`            | INTEGER   | Primary key                          |
| `discord_id`    | TEXT      | Unique, not null                     |
| `username`      | TEXT      | Discord display name at login time   |
| `session_token` | TEXT      | Unique, nullable (cleared on logout) |
| `has_voted`     | BOOLEAN   | Default false                        |

### `ballots`

| Column     | Type    | Notes                                          |
| ---------- | ------- | ---------------------------------------------- |
| `id`       | INTEGER | Primary key                                    |
| `user_id`  | INTEGER | Nullable. Null for paper ballots               |
| `source`   | TEXT    | `"online"` or `"paper"`                        |
| `rank_1`   | INTEGER | Song ID, not null                              |
| `rank_2`   | INTEGER | Song ID, not null                              |
| `rank_3`   | INTEGER | Song ID, not null                              |
| `rank_4`   | INTEGER | Song ID, not null                              |
| `rank_5`   | INTEGER | Song ID, not null                              |

Ballots are immutable once written. The `users.has_voted` flag is set in the same transaction as the online ballot insert.

## 6. Routes

| Method | Path                       | Auth | Purpose                                                                  |
| ------ | -------------------------- | ---- | ------------------------------------------------------------------------ |
| GET    | `/`                        | No   | Landing page with "Login with Discord" button. Always renders the same page regardless of session state. |
| GET    | `/auth/discord`            | No   | Redirect to the Discord OAuth authorize URL.                             |
| GET    | `/auth/discord/callback`   | No   | Exchange the authorization code, verify guild membership, create session, redirect to `/vote`. |
| GET    | `/vote`                    | Yes  | Render the voting page. Returns 401 if no valid session.                 |
| POST   | `/vote`                    | Yes  | Accept a ballot submission. Returns 401 if no valid session.             |
| GET    | `/results`                 | No   | Render STV results. Empty until the tally has been run.                  |
| POST   | `/logout`                  | Yes  | Clear the session and redirect to `/`.                                   |

A single auth middleware wraps `/vote` (both methods) and `/logout`. It reads the session cookie, looks up `users.session_token`, and rejects with 401 (rendered as a small page with a link back to `/`) if the session is missing or invalid. All other routes are public.

## 7. Authentication

```
User                  App                            Discord
 │                     │                              │
 ├── GET / ──────────► │ render landing               │
 │                     │                              │
 ├── click Login ────► │ GET /auth/discord            │
 │                     ├── 302 ─────────────────────► │ authorize URL
 │ ◄───────────────────┤◄── user authorizes ──────────┤
 │                     │  GET /auth/discord/callback  │
 │                     ├── POST token exchange ─────► │
 │                     │ ◄─ access token ───────────  │
 │                     ├── GET /users/@me/guilds ───► │
 │                     │ ◄─ guild list ─────────────  │
 │                     │ verify guild ∈ list          │
 │                     │ upsert user, create session  │
 │ ◄── 302 /vote ──────┤ set Set-Cookie               │
```

- Required scopes: `identify`, `guilds`. (`guilds` returns the list of guilds the user belongs to; this is used to verify membership in `DISCORD_GUILD_ID`.)
- Session cookies are HttpOnly, Secure, SameSite=Lax. The cookie value is a random token stored in `users.session_token`.
- An already-logged-in user navigating to `/` simply sees the landing page again. There is no "already logged in" branch on `/`.
- Sessions do not expire on a timer for this single-election use case; they live until logout or until the database is wiped after the election.

## 8. Frontend

### Templates

Three templates rendered server-side via `html/template`:

- `landing.html` — title and "Login with Discord" button.
- `vote.html` — two columns (song pool on the left, "Your Top 5" slots on the right), submit button. After submission, the page renders a read-only "your vote is locked in" view showing the user's ranking.
- `results.html` — ranked table of the top 10 once the tally has been run.

Templates are parsed once at startup and reused for every request.

### JavaScript

Vanilla, no libraries. Approximately 60–80 lines, served from `web/static/vote.js`.

Interaction model is **click-to-select**, not HTML5 drag-and-drop (drag-and-drop is awkward on mobile):

- Click a song in the left column → it moves to the next open slot on the right.
- Click a song in the right column → it returns to the left pool.
- Up/down arrows on each right-column entry to reorder.
- Submit is disabled until exactly 5 songs are selected.
- On submit, POST `application/json` to `/vote` with `{"ranks": [<id>, <id>, <id>, <id>, <id>]}`. On success the server returns `204 No Content` and the client navigates to `/vote`, which renders the locked view. On `409 Conflict` (already voted) the client also navigates to `/vote` to surface the locked view; other 4xx responses are shown to the user as an error.

Each song row renders its ID, title, artist, and small icon links to its YouTube and Spotify pages. URLs are part of the hardcoded song data.

## 9. CLI

```
benefitshow serve              # start the web server (default :8080)
benefitshow import <path.csv>  # import paper ballots from a CSV file
benefitshow tally              # run the STV algorithm, print and persist results
```

CSV format: header optional, 5 integer columns per row, no `user_id` or `source` columns. The importer inserts each row as a `paper` ballot with `user_id = NULL`.

## 10. Tally: STV with the Droop Quota

Single Transferable Vote, multi-winner. For `N` seats and `V` valid ballots, the **Droop quota** is:

```
quota = floor(V / (N + 1)) + 1
```

Algorithm:

1. First-preference counts: each ballot contributes 1 vote to its current top choice.
2. Any candidate at or above the quota is **elected**. Their surplus (`votes - quota`) is transferred to the remaining preferences on their ballots, weighted proportionally.
3. If no candidate reaches the quota and seats remain, the candidate with the fewest votes is **eliminated** and their ballots are transferred at full weight.
4. Repeat until `N` candidates have been elected, or until the remaining candidates fill the remaining seats.

For this election, `N = 10`. After the iterative tally, surviving candidates are ranked by election order; eliminated candidates are appended in reverse elimination order so the result page can show a full ordering.

Ties (equal vote counts at elimination time) are broken deterministically by song ID ascending. This is documented but unlikely to matter at this voter count.

## 11. Configuration

Configuration is read from environment variables on startup by the `internal/config` package, which validates required variables and exposes a typed struct used by the rest of the app:

| Variable                | Required | Purpose                                     |
| ----------------------- | -------- | ------------------------------------------- |
| `DISCORD_CLIENT_ID`     | yes      | OAuth app client ID                         |
| `DISCORD_CLIENT_SECRET` | yes      | OAuth app client secret                     |
| `DISCORD_GUILD_ID`      | yes      | Guild whose members may vote                |
| `DISCORD_REDIRECT_URI`  | yes      | Must match the Discord app's registered URI |
| `SESSION_SECRET`        | yes      | Random bytes used when generating tokens    |
| `PORT`                  | no       | Listen port (default `8080`)                |
| `DB_PATH`               | no       | SQLite file path (default `./benefitshow.db`) |

The binary refuses to start the `serve` subcommand if any required variable is missing.

## 12. Project Layout

```
benefitshow/
├── cmd/benefitshow/main.go   entrypoint, CLI subcommand dispatch
├── internal/
│   ├── config/               env var loading and validation
│   ├── server/               HTTP handlers, router, middleware
│   ├── auth/                 Discord OAuth, session helpers
│   ├── ballot/               ballot read/write, CSV import
│   ├── stv/                  STV tally
│   └── songs/                hardcoded song list and types
├── web/
│   ├── templates/            *.html (embedded via go:embed)
│   └── static/               style.css, vote.js (embedded)
├── Makefile                  build, run, lint, deploy
├── benefitshow.service       systemd unit
├── go.mod
└── go.sum
```

The SQLite database file is created at runtime in the working directory and is not checked in.

## 13. Deployment

The service runs on a small VM (e.g. GCP `e2-micro`) behind Caddy, which auto-provisions and renews a Let's Encrypt certificate.

### One-time VM setup

1. Provision the VM and open ports 80 and 443.
2. Point a domain at the VM's external IP via an A record.
3. Install Caddy and configure a single site directive that reverse-proxies to `localhost:8080`.
4. Create `/opt/benefitshow` (owned by the deploy user) for the binary, and `/var/lib/benefitshow` (owned by the service user, mode `0750`) for the SQLite database. Splitting code from state lets the service run as an unprivileged `nobody:nogroup` user without making the deploy target world-writable.
5. Install `benefitshow.service` into `/etc/systemd/system/` and `systemctl enable` it.
6. Register the OAuth app on the Discord developer portal, copy client ID/secret, and add the production callback URL.
7. Populate `/etc/benefitshow.env` with `DB_PATH=/var/lib/benefitshow/benefitshow.db` plus the Discord and session variables. `chmod 600`.

### Repeatable deploys

```
make deploy
```

Cross-compiles for Linux, copies the binary to the VM, and restarts the service.

### Systemd unit (sketch)

```ini
[Unit]
Description=Benefit Show Voting App
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
WorkingDirectory=/opt/benefitshow
ExecStart=/opt/benefitshow/benefitshow serve
Restart=on-failure
EnvironmentFile=/etc/benefitshow.env

[Install]
WantedBy=multi-user.target
```

## 14. Security Considerations

- **Auth scope is the guild check.** Any Discord user in the configured guild can vote. There is no further role or permission check.
- **One ballot per user** is enforced at the application layer (`users.has_voted`) and via a uniqueness contract (only one online ballot per `user_id`). Paper ballots have `user_id = NULL` and are not de-duplicated by the app — the administrator is responsible for not entering the same paper ballot twice.
- **Session cookies** are HttpOnly, Secure, SameSite=Lax. The session token is a random opaque value, not a JWT.
- **CSRF** for `POST /vote` and `POST /logout` is mitigated by SameSite=Lax cookies and the requirement that POST `/vote` carry a JSON content type. No additional CSRF token is issued.
- **OAuth state parameter** is set on `/auth/discord` and verified on callback to prevent login CSRF.
- **No PII** beyond Discord ID and display name is stored. Ballots are not linked back to users on the results page.
