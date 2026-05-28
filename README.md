# ghstats

Dynamically generated GitHub profile cards, rendered server-side as SVG. A focused
re-implementation of [github-readme-stats](https://github.com/anuraghazra/github-readme-stats)
covering three cards: **most used languages**, **contribution streaks**, and **total
contributions** (plus stats / pinned-repo / gist cards that ship alongside).

Built entirely in **Go + [templ](https://templ.guide) + HTMX + Tailwind**, packaged as a
container, and deployed on **AWS App Runner**.

---

## How it works

```
browser / README ──HTTP──> Go server ──> in-memory TTL cache ──miss──> GitHub GraphQL+REST
                              │
                              └─> templ renders an SVG ──> served with Cache-Control headers
```

1. A request hits an endpoint like `/api/streak?username=octocat`.
2. The server checks an **in-memory TTL cache** keyed by the full request URI. On a hit it
   returns the cached SVG immediately.
3. On a miss it calls the **GitHub API** (GraphQL for most data, REST for all-time commit
   counts), aggregates the result, and renders an SVG using a **templ** component.
4. The SVG is cached and returned with `Cache-Control` headers so a CDN (or GitHub's image
   proxy / Camo) can cache it too.

### Rate limiting & tokens

GitHub allows 5,000 req/hr **per token** (60/hr unauthenticated). The client (`internal/github/client.go`)
rotates round-robin across every token you provide, so adding more tokens multiplies the
budget. Set them via environment variables:

- `GITHUB_TOKEN` — a single personal access token, **or**
- `PAT_1`, `PAT_2`, … — a numbered series, all used together.

A classic PAT with **no scopes** (public data only) is enough for public profiles. Without a
token the app still runs but is quickly rate-limited.

### Caching

Responses set `max-age` / `s-maxage` / `stale-while-revalidate`. The `cache_seconds` query
param tunes it, clamped to **30 min … 24 h** (default 4 h) to protect the rate limit.

---

## Endpoints

| Endpoint | Purpose | Required params |
|---|---|---|
| `GET /` | HTMX + Tailwind playground UI | — |
| `GET /api/streak` | Total contributions, current & longest streak | `username` |
| `GET /api/top-langs` | Most used languages bar list | `username` |
| `GET /api` | Stats summary card (stars, commits, PRs, rank…) | `username` |
| `GET /api/pin` | Single pinned repository | `username`, `repo` |
| `GET /api/gist` | Single gist | `id` |
| `GET /healthz` | Liveness probe | — |

### Common query params (all cards)

| Param | Default | Notes |
|---|---|---|
| `theme` | `default` | See themes below |
| `title_color`, `icon_color`, `text_color`, `bg_color`, `border_color` | from theme | Raw hex, no `#` needed |
| `hide_border`, `hide_title` | `false` | |
| `show_icons` | `false` | Stats card icons |
| `custom_title` | — | Overrides the card title |
| `card_width` | per-card | Pixels |
| `hide` | — | CSV of rows to hide (stats card), e.g. `hide=issues,prs` |
| `cache_seconds` | `14400` | Clamped 1800–86400 |

Card-specific: `top-langs` takes `langs_count` (default 5); `api` (stats) takes
`include_all_commits`, `count_private`, `hide_rank`.

### Example

```markdown
![streak](https://YOUR_HOST/api/streak?username=octocat&theme=tokyonight)
![langs](https://YOUR_HOST/api/top-langs?username=octocat&langs_count=8&theme=tokyonight)
```

### Themes

`default`, `dark`, `transparent`, `radical`, `merko`, `gruvbox`, `tokyonight`, `onedark`,
`cobalt`, `synthwave`, `dracula`, `nord`, `vue`, `github_dark`, `catppuccin_latte`,
`catppuccin_mocha`, `rose_pine`. (Defined in `internal/themes/themes.go`.)

---

## Project layout

```
cmd/server/         entrypoint: loads tokens, starts HTTP server
internal/server/    routing, caching, query parsing, error mapping
internal/github/    GitHub API client + per-feature fetchers (stats, langs, streak, repo, gist)
internal/cards/     templ SVG components + view-model builders
internal/themes/    color palettes
internal/render/    server-side text measurement (for wrapping/centering)
internal/web/        HTMX + Tailwind playground page and /preview fragment
internal/cache/      concurrency-safe in-memory TTL cache
```

`*.templ` files compile to `*_templ.go` via `templ generate` (these generated files are not
committed — they're produced by the build).

---

## Local development

**Prerequisites:** Go 1.25+. (templ itself is run pinned via `go run`, nothing to install.)

```bash
export GITHUB_TOKEN=ghp_yourtoken   # optional but strongly recommended
make run                            # templ generate + go run, listens on :8080
```

Open <http://localhost:8080> for the playground, or hit an endpoint directly:
<http://localhost:8080/api/streak?username=octocat>.

### Make targets

| Target | Does |
|---|---|
| `make run` | Generate templ + run the server |
| `make build` | Generate templ + build a static binary to `bin/ghstats` |
| `make generate` | Render `*.templ` → `*_templ.go` (pinned templ version) |
| `make test` / `make vet` / `make fmt` | Standard checks |
| `make docker-build` | Build the container image |
| `make docker-push` | Build, ECR-login, tag, push (needs AWS vars) |
| `make deploy` | Push image + trigger an App Runner deployment |

> **Frontend note:** Tailwind and HTMX load from their CDNs in `internal/web/pages.templ`,
> so there's no CSS/JS build step. To self-host Tailwind for production, install the Tailwind
> standalone CLI and follow `make css`.

---

## Deployment — AWS App Runner

The app is a stateless container listening on `:8080` (`PORT` overrides). App Runner pulls an
image from ECR, runs it, gives you an HTTPS URL, and autoscales.

### One-time setup

1. **Create the ECR repository:**
   ```bash
   aws ecr create-repository --repository-name ghstats --region us-east-1
   ```

2. **Build & push the first image** (so the service has something to start from):
   ```bash
   make docker-push \
     AWS_ACCOUNT_ID=123456789012 \
     AWS_REGION=us-east-1
   ```

3. **Create the App Runner service** (Console → App Runner → Create service):
   - Source: **Container registry → Amazon ECR**, pick the `ghstats:latest` image.
   - Port: **8080**.
   - Environment variable: `GITHUB_TOKEN` = your PAT (store as a secret).
   - Health check path: `/healthz`.
   - Note the service ARN it creates.

### Continuous deployment (GitHub Actions)

`.github/workflows/deploy.yml` builds, pushes to ECR, and triggers a new App Runner
deployment on every push to `main`. It stays dormant until you configure it.

1. Create an IAM role for **GitHub OIDC** trusting `token.actions.githubusercontent.com`,
   with permissions for `ecr:GetAuthorizationToken`, ECR push, and `apprunner:StartDeployment`.
2. Add these **repository Variables** (Settings → Secrets and variables → Actions → Variables):
   | Variable | Value |
   |---|---|
   | `AWS_REGION` | e.g. `us-east-1` |
   | `AWS_ROLE_ARN` | the OIDC role ARN |
   | `ECR_REPO` | `ghstats` |
   | `APPRUNNER_SERVICE_ARN` | the service ARN from step 3 above |

Once `APPRUNNER_SERVICE_ARN` is set, the deploy workflow activates automatically.

### CI

`.github/workflows/ci.yml` runs `templ generate` → `go vet` → `go build` → `go test` on every
push and pull request.

---

## Configuration reference

| Env var | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | Listen port |
| `GITHUB_TOKEN` | — | Single GitHub PAT |
| `PAT_1`, `PAT_2`, … | — | Multiple PATs, rotated for more rate-limit budget |
