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

### Single-user mode

By default the app serves cards for any username. Set **`ALLOWED_USERNAME`** to lock an
instance to one GitHub user — this is the intended way to **host your own**:

- When set, the `username` param becomes optional; if omitted it defaults to the allowed
  user, and any non-matching `username` is rejected with an error card.
- The playground UI pre-fills and locks the username field.
- Leave it unset (or empty) to run an open, multi-user instance.

```bash
export ALLOWED_USERNAME=michael-duren
```

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

When `ALLOWED_USERNAME` is set, `username` is optional on the endpoints above (it defaults to
the allowed user) and a mismatched value is rejected.

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
export GITHUB_TOKEN=ghp_yourtoken      # optional but strongly recommended
export ALLOWED_USERNAME=michael-duren  # optional — lock to one user (host your own)
make run                               # templ generate + go run, listens on :8080
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

Infrastructure lives in `infra/` (Terraform/OpenTofu) and the deploy steps are wrapped by
the scripts in `scripts/`:

| Script | Purpose |
|---|---|
| `scripts/create-s3.sh` | Bootstrap the remote-state S3 bucket + write `infra/backend.hcl` (run once) |
| `scripts/bootstrap.sh` | First-time deploy: init backend → create ECR → push first image → create App Runner |
| `scripts/deploy.sh` | Manual redeploy: build + push a new image, trigger a deployment |
| `scripts/destroy.sh` | Tear down all Terraform-managed resources (confirms first) |

Every script takes `AWS_REGION` (default `us-east-1`) and `APP` (default `ghstats`) as
environment overrides.

### Prerequisites

1. **AWS CLI** configured (`aws sts get-caller-identity` returns your account).
2. **Docker** working without sudo in your shell — `bootstrap.sh`/`deploy.sh` build the
   image. (If you just added yourself to the `docker` group, run them from a `newgrp docker`
   shell until your next login.)
3. **OpenTofu** (`sudo pacman -S opentofu`, CLI `tofu`) or Terraform from the AUR. The scripts
   call `tofu`; alias or edit if you use `terraform`.
4. Make the scripts executable once: `chmod +x scripts/*.sh`.

### First-time deploy (step by step)

```bash
# 1. Configure variables. Set allowed_username (lock to your GitHub user) and
#    github_repo (enables the CI deploy role). The token is passed via env so it
#    never lands in a committed file.
cp infra/terraform.tfvars.example infra/terraform.tfvars
$EDITOR infra/terraform.tfvars
export TF_VAR_github_token=ghp_yourtoken

# 2. Create the S3 bucket that holds Terraform state. This must exist before
#    `tofu init` can use it, so it's done with the AWS CLI, not Terraform. The
#    script also writes infra/backend.hcl with the bucket name (which includes
#    your account id, keeping it globally unique).
scripts/create-s3.sh

# 3. Deploy. bootstrap.sh runs the ordered flow that a single `tofu apply` can't:
#    App Runner refuses to create a service pointing at an image tag that doesn't
#    exist yet, so the repo is created and an image pushed *before* the service.
#      init backend -> tofu apply -target ECR -> make docker-push -> tofu apply
scripts/bootstrap.sh
```

`bootstrap.sh` prints the public HTTPS URL at the end; you can re-read it anytime with:

```bash
cd infra && tofu output -raw service_url
```

State is stored remotely in S3 (`backend "s3"` in `infra/providers.tf`, configured via the
generated `infra/backend.hcl`) with S3-native locking — no DynamoDB table required.

### Shipping changes after the first deploy

```bash
scripts/deploy.sh        # build + push a fresh image, then trigger a deployment
```

This is the manual equivalent of the GitHub Actions workflow below — use it for one-off
releases or before CI is wired up. To tear everything down (leaves the state bucket):

```bash
scripts/destroy.sh
```

### Continuous deployment (GitHub Actions)

`.github/workflows/deploy.yml` builds, pushes to ECR, and triggers a new App Runner
deployment on every push to `main`. With `github_repo` set in your tfvars, Terraform creates
the OIDC role for it. Wire it up by adding these **repository Variables** (Settings → Secrets
and variables → Actions → Variables), reading the values from `tofu output`:

| Variable | Source |
|---|---|
| `AWS_REGION` | your region, e.g. `us-east-1` |
| `AWS_ROLE_ARN` | `tofu output github_ci_role_arn` |
| `ECR_REPO` | `ghstats` |
| `APPRUNNER_SERVICE_ARN` | `tofu output apprunner_service_arn` |

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
| `ALLOWED_USERNAME` | — | Locks the instance to one GitHub username (empty = serve any) |
