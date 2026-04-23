# CommitLens

A small tool to aggregate **merged PRs**, **commits**, and **additions/deletions** per contributor on GitHub. Ships with a **terminal TUI** and an **embedded web UI** (Vite + React) inside the same binary.

**[简体中文说明 →](./README-zh.md)**

## Features

- Fetches merged PRs, per-PR commit lists, and diff stats via the GitHub API; results are cached locally.
- **Contributor table**: PR count, commit count, lines added/removed.
- **PR trends** by week / month / quarter / year; one repo or a multi-select set.
- **Co-authored-by** trailers in any PR commit message are parsed (`Co-authored-by: … <email>`). Only `users.noreply.github.com` addresses (including `id+username@` form) are turned into a GitHub login. Each person is counted at most once per PR. Primary author and co-authors are all credited for **PR count, commit count, and add/del** (totals can exceed “unique repo lines” when a PR is shared, by design).

## Screenshots

Examples use repo **`kubeovn/kube-ovn`**; your data and granularity will differ.

### Terminal TUI

Single-repo view, **monthly** merge-PR trend plus per-contributor bars (left list + right chart, horizontal scroll for long ranges):

![CommitLens TUI: kube-ovn monthly org + contributor PR trends](docs/screenshots/tui-kube-ovn-monthly-pr-trends.png)

### Web

**By week** — top: repo-wide bar chart with PR counts; below: one row per contributor. Bottom slider scrubs the time range.

![CommitLens Web: kube-ovn weekly](docs/screenshots/web-kube-ovn-weekly-pr-trends.png)

**By quarter** — same layout, quarter-based axis.

![CommitLens Web: kube-ovn quarterly](docs/screenshots/web-kube-ovn-quarterly-pr-trends.png)

## Build

Requires **Go** (see `go.mod`) and **Node.js** (to bundle the web UI).

```bash
make build   # runs frontend build, then go build -> ./commitlens
```

or:

```bash
cd frontend && npm install && npm run build
go build -o commitlens .
```

## Configuration

Copy the sample and adjust:

```bash
cp config.example.yaml ~/.commitlens/config.yaml
```

| Key | Notes |
|-----|--------|
| `github.token` | Optional; if empty, `gh auth token` is used when available. |
| `repositories` | List of `owner` / `repo` pairs to aggregate. |
| `cache.dir` | Directory for raw PR data and derived stats. |
| `web.port` | Web mode port (overridable with `--port` on the CLI). |

## Usage

**TUI (default):**

```bash
./commitlens --config /path/to/config.yaml
# or
make run
```

**Web UI:**

```bash
./commitlens --web --port 8080 --config /path/to/config.yaml
# or
make run-web
```

The first run syncs from GitHub; you can refresh from the TUI or the web. Aggregation and co-author parsing live under `internal/stats` (see `coauthor.go`).

## Development

```bash
make test
cd frontend && npm run lint
```

## Project layout (partial)

- `cmd/` — CLI, sync, wiring
- `internal/github/` — GitHub client
- `internal/stats/` — aggregation + `Co-authored-by`
- `internal/tui/` — Bubble Tea TUI
- `internal/web/` — HTTP API + static frontend
- `frontend/` — Vite + React, embedded with `//go:embed`
