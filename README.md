# benefitshow

A small ranked-choice voting app for picking a 5-song setlist from a pool of ~30 candidates. Voters log in with Discord (gated by membership in a configured guild) and rank their top 5 picks. After voting closes, paper ballots are imported via CSV and a Single Transferable Vote (STV) tally produces a ranked top 10.

Built as a single Go binary serving its own templates and static assets — no JavaScript framework, no separate frontend.

## Status

Pre-alpha. Active build target: weekend of 2026-05-03. See [roadmap.md](roadmap.md) for milestones.

## Stack

- **Backend:** Go (stdlib `net/http` + `html/template`)
- **Storage:** SQLite (single file, embedded)
- **Auth:** Discord OAuth2 with guild membership check
- **Frontend:** Vanilla HTML/CSS/JS, click-to-rank UI
- **Deploy:** Single binary on an Ubuntu VM behind Caddy (auto-HTTPS), managed by `systemd`

## Documentation

- [spec.md](spec.md) — full design: data model, routes, auth flow, STV algorithm, deployment
- [roadmap.md](roadmap.md) — milestone-by-milestone work breakdown

## Local development

```sh
make run                      # build and start the server on :8080
make build                    # build the binary
make lint                     # go vet
./benefitshow import b.csv    # import paper ballots
./benefitshow tally           # run STV, print top 10
```

Required environment variables for the `serve` subcommand:

| Variable                | Purpose                                   |
| ----------------------- | ----------------------------------------- |
| `DISCORD_CLIENT_ID`     | OAuth app client ID                       |
| `DISCORD_CLIENT_SECRET` | OAuth app client secret                   |
| `DISCORD_GUILD_ID`      | Guild whose members are allowed to vote   |
| `DISCORD_REDIRECT_URI`  | OAuth callback URL                        |
| `SESSION_SECRET`        | Random string used to sign session tokens |

## License

MIT — see [LICENSE](LICENSE).
