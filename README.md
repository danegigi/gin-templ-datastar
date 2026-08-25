# gin-templ-datastar

A server-rendered Go admin panel built with **Gin**, **templ**, **Datastar**, and **Tailwind CSS** — a single self-contained binary that renders HTML on the server, layers in reactive UI with zero hand-written JavaScript, and exposes a JSON API secured with JWT.

It's a complete, working example of the "hypermedia-driven Go" stack: type-safe templates, raw SQL (no ORM), skeleton-first lazy loading, and parallel database queries with goroutines.

## 📘 Tutorial

This project ships with a full written tutorial for **beginner-to-mid-level developers** that explains not just *what* the code does but *why* it's organized the way it is — walking through `main.go`, models, stores, routes, handlers, templ pages, layouts, middleware, authentication, Datastar, Tailwind, air, the Makefile, the skeleton-first performance work, and end-to-end testing with Playwright.

- **Read it as a web page:** [`docs/index.html`](docs/index.html) (also published via GitHub Pages)
- **Read the source:** [`TUTORIAL.md`](TUTORIAL.md)

Regenerate the HTML after editing the markdown:

```bash
npm install        # once, installs the `marked` renderer
node scripts/build-tutorial.js
```

---

## Table of contents

1. [Stack](#stack)
2. [Project structure](#project-structure)
3. [Prerequisites](#prerequisites)
4. [Environment variables](#environment-variables)
5. [Local setup](#local-setup)
6. [Build system (Makefile)](#build-system-makefile)
7. [Live reload with air](#live-reload-with-air)
8. [Running the server](#running-the-server)
9. [Routes reference](#routes-reference)
10. [Authentication](#authentication)
11. [Database layer](#database-layer)
12. [Data models](#data-models)
13. [Middleware](#middleware)
14. [Views (templ + Datastar)](#views-templ--datastar)
15. [Performance: skeleton-first loading + parallel queries](#performance-skeleton-first-loading--parallel-queries)
16. [Testing](#testing)
17. [Known limitations](#known-limitations)

---

## Stack

| Layer | Technology | Version |
|---|---|---|
| Language | Go | 1.26+ |
| HTTP router | [Gin](https://github.com/gin-gonic/gin) | v1.10 |
| Server-rendered HTML | [templ](https://templ.guide) | v0.3.1020 |
| Reactive UI | [Datastar](https://data-star.dev) | CDN (SSE-based signals) |
| Styling | [Tailwind CSS](https://tailwindcss.com) | v3 |
| Database | MySQL via [sqlx](https://github.com/jmoiron/sqlx) + `database/sql` | sqlx v1.4 |
| JWT | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) | v5 |
| Session cookies | [gin-contrib/sessions](https://github.com/gin-contrib/sessions) (Gorilla backend) | v1.0 |
| Billing | [stripe-go](https://github.com/stripe/stripe-go) | v82 |
| Env loading | [godotenv](https://github.com/joho/godotenv) | v1.5 |
| Parallel queries | [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) | v0.22 |
| Live reload | [air](https://github.com/air-verse/air) | — |

---

## Project structure

```
.
├── cmd/
│   └── server/
│       └── main.go                   Entry point – wires DB, stores, handlers, Gin engine
│
├── internal/
│   ├── http/
│   │   └── handlers/
│   │       ├── deps.go               Deps struct injected into all handlers
│   │       ├── routes.go             RegisterRoutes() – all HTML + /api/* routes
│   │       ├── auth_handler.go       GetLogin, PostLogin, Logout, APILogin, generateJWT
│   │       ├── home_handler.go       Dashboard shell + lazy-loaded fragment endpoints
│   │       ├── accounts_handler.go   Accounts list/detail (shell + fragments), toggles, activity
│   │       ├── affiliates_handler.go USPS reps + affiliates (HTML + API)
│   │       ├── labels_handler.go     Labels report (parallel queries), APIGetLabel, APIGetTotalBilled
│   │       └── settings_handler.go   Countries list, toggle active
│   │
│   ├── middleware/
│   │   ├── auth.go                   RequireAuth() (JWT), SessionAuth() (cookie)
│   │   ├── cors.go                   CORS() – configurable origin whitelist
│   │   └── session.go                Sessions(), LoadSessionAdmin(), SetSessionAdmin(), ClearSession()
│   │
│   ├── models/
│   │   └── models.go                 Plain Go structs (Admin, User, Order, OrderLabel, Country, …)
│   │
│   ├── store/                        All SQL lives here (raw sqlx, no ORM)
│   │   ├── db.go                     Connection pool setup
│   │   ├── admin_store.go
│   │   ├── user_store.go
│   │   ├── order_store.go
│   │   └── label_store.go
│   │
│   ├── services/                     Placeholder – extract business logic here as it grows
│   │
│   └── views/
│       ├── layouts/
│       │   └── base.templ            HTML shell, sidebar, topbar, skeleton/lazy-load helpers
│       └── pages/
│           ├── login.templ           Login page
│           ├── home.templ            Dashboard shell + stat/table fragments
│           ├── accounts.templ        Accounts list with Datastar live search
│           ├── account_detail.templ  Account detail (shell + info/activity fragments)
│           ├── affiliates.templ      USPS reps + affiliates list/create/detail
│           └── misc.templ            Labels report, deactivated, favorites, settings/countries
│
├── static/
│   ├── css/
│   │   ├── input.css                 Tailwind entry file
│   │   └── app.css                   Generated Tailwind output
│   ├── images/                       Favicon, logos
│   └── js/                           Optional custom JS
│
├── docs/
│   ├── index.html                    Tutorial, published via GitHub Pages
│   └── .nojekyll                     Serve as-is (no Jekyll processing)
│
├── scripts/
│   └── build-tutorial.js             Renders TUTORIAL.md → TUTORIAL.html + docs/index.html
│
├── .air.toml                         air live-reload config
├── .env.example                      Environment variable template
├── go.mod                            Module: github.com/grandshipper/admin-v2
├── Makefile                          Build targets
├── package.json                      Node deps (Tailwind CSS + tutorial renderer)
├── tailwind.config.js
├── TUTORIAL.md                       The full written tutorial
└── TUTORIAL.html                     Rendered tutorial (generated)
```

> Note: the Go module path is `github.com/grandshipper/admin-v2` (in `go.mod`). Rename it with a find-and-replace across imports if you fork this for a different project.

---

## Prerequisites

| Tool | Install |
|---|---|
| Go 1.21+ | https://go.dev/dl |
| templ CLI | `go install github.com/a-h/templ/cmd/templ@latest` |
| Node.js ≥ 18 | https://nodejs.org (Tailwind CSS build + tutorial renderer) |
| MySQL | A running instance for the app's data |
| air (optional) | `go install github.com/air-verse/air@latest` |

Verify:

```bash
go version        # go1.26.x or newer
templ --version   # v0.3.x
node --version    # v18+
```

---

## Environment variables

Copy `.env.example` to `.env` and fill in the values:

```bash
cp .env.example .env
```

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | ✅ | MySQL DSN. Format: `user:pass@tcp(host:port)/dbname?parseTime=true` |
| `READ_DATABASE_URL` | optional | Explicit read-replica DSN (falls back to `DATABASE_URL`) |
| `WRITE_DATABASE_URL` | optional | Explicit write-primary DSN (falls back to `READ_DATABASE_URL`) |
| `PORT` | optional | HTTP listen port. Default: `8080` |
| `JWT_KEY` | ✅ | HMAC secret used to sign and verify JWTs |
| `JWT_EXPIRATION` | optional | Token lifetime. Default: `8h` |
| `SESSION_SECRET` | ✅ | 32-byte random string for encrypting session cookies |
| `CORS_ORIGIN` | optional | Comma-separated allowed origins. Supports a `regexp:` prefix |
| `STRIPE_SECRET` | optional | Stripe secret key — required only for `/api/get-total-billed/:customerId` |
| `AWS_REGION` / `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `S3_BUCKET` | optional | Required only if label PDFs are stored on S3 |

> **Security:** never commit a `.env` with real secrets. `.env` is in `.gitignore`; commit only `.env.example` with placeholders.

---

## Local setup

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env – at minimum set DATABASE_URL, JWT_KEY, SESSION_SECRET

# 2. Install Node deps (Tailwind + tutorial renderer)
npm install

# 3. Install Go deps
go mod download

# 4. Generate templ code, build Tailwind CSS, compile the binary
make all

# 5. Run
./bin/admin-v2
# → admin-v2 listening on :8080
```

Open http://localhost:8080/login in your browser.

---

## Build system (Makefile)

```bash
make all        # generate → css → build  (default)
make generate   # run templ code generation (*.templ → *_templ.go)
make css        # rebuild Tailwind CSS (static/css/input.css → static/css/app.css)
make build      # compile Go binary → bin/admin-v2
make run        # all + start binary
make dev        # generate + css + go run ./cmd/server (no binary written)
make test       # go test ./...
make tidy       # go mod tidy
make clean      # remove bin/, app.css, all *_templ.go files
make help       # print all targets
```

**Important:** always run `make generate` after editing any `.templ` file. The `*_templ.go` files are generated outputs — don't edit them directly.

---

## Live reload with air

[air](https://github.com/air-verse/air) watches source files and automatically re-runs `make generate css`, recompiles, and restarts the server on every save.

```bash
go install github.com/air-verse/air@latest   # once
air                                            # from the project root
```

The `.air.toml` config:

- Watches `.go`, `.templ`, `.css`, `.env` files under `cmd/`, `internal/`, `static/`
- Excludes `node_modules/`, `.air_tmp/`, `bin/`, `vendor/`, and `*_templ.go` (to avoid double-rebuild loops)
- Runs `make generate css` as a pre-build step
- Stops the watcher loop on build errors (`stop_on_error = true`)

---

## Running the server

```bash
make all && ./bin/admin-v2   # production build
make dev                     # development (go run, no binary written)
air                          # live reload
```

The server binds to `0.0.0.0:<PORT>` (default `:8080`). Static files are served from `./static/`.

---

## Routes reference

### HTML routes (session auth)

Routes under the protected group require a valid session cookie (set on `POST /login`). Unauthenticated requests are redirected to `/login`.

| Method | Path | Description |
|---|---|---|
| GET | `/ping`, `/healthcheck` | Health check |
| GET / POST | `/login` | Login page / submit credentials |
| GET | `/logout` | Clear session |
| GET | `/` | Dashboard (skeleton shell; sections lazy-load) |
| GET | `/dashboard/stat/*`, `/dashboard/table/*` | Dashboard fragment endpoints |
| GET | `/accounts` | Accounts list (shell + Datastar live search) |
| GET | `/accounts/search` | Accounts table fragment (debounced search) |
| GET | `/accounts/:id` | Account detail shell |
| GET | `/accounts/:id/info`, `/accounts/:id/activities` | Account detail fragments |
| POST | `/accounts/:id/change-password` | Change password |
| POST | `/accounts/:id/favorite` | Toggle favorite |
| POST | `/accounts/:id/suspend` | Toggle suspend |
| POST | `/accounts/:id/activity` | Add admin note |
| POST | `/accounts/:id/restore` | Restore soft-deleted account |
| GET | `/favorites` | Favorited accounts |
| GET | `/usps-reps`, `/usps-reps/list` | USPS reps list (shell + fragment) |
| GET / POST | `/usps-reps/create` | Create USPS rep |
| GET | `/usps-reps/:id` | Rep detail shell |
| POST | `/usps-reps/:id/change-password` | Change rep password |
| GET | `/affiliates`, `/affiliates/list` | Affiliates list (shell + fragment) |
| GET / POST | `/affiliates/create` | Create affiliate |
| GET | `/affiliates/:id` | Affiliate detail shell |
| POST | `/affiliates/:id/change-password` | Change affiliate password |
| GET | `/deactivated`, `/deactivated/list` | Deactivated accounts (shell + fragment) |
| GET / POST | `/labels`, `/labels/report` | Labels report form / run report |
| GET | `/settings` | Settings — countries |
| POST | `/settings/countries/:id/toggle` | Toggle country active flag |

### JSON API routes (JWT auth)

All `/api/*` routes require a JWT passed as `x-auth-token: <token>` or `Authorization: Bearer <token>`.

| Method | Path |
|---|---|
| POST | `/api/login` |
| POST | `/api/register-rep` |
| POST | `/api/account/change-password` |
| POST | `/api/rep/change-password` |
| POST | `/api/affiliate/create` |
| POST | `/api/affiliate/change-password` |
| POST | `/api/reports/print-performance` |
| POST | `/api/reports/active-users-performance` |
| POST | `/api/reports/labels` |
| POST | `/api/reports/new-labels` |
| POST | `/api/reports/labels-by-marketplace` |
| POST | `/api/get-label` |
| GET | `/api/get-total-billed/:customerId` |
| POST | `/api/update-pb-status` |

---

## Authentication

Two independent auth layers coexist on the same binary:

**Session auth (HTML routes).** On `POST /login`, the handler verifies the bcrypt password hash and stores the admin's ID in an encrypted Gorilla cookie session. `LoadSessionAdmin()` restores it into the Gin context on every request; `SessionAuth()` gates protected routes and redirects to `/login` when absent. Cookies are `HttpOnly` and `Secure` in production.

**JWT auth (API routes).** On `POST /api/login`, credentials are validated and a JWT is issued (HS256, signed with `JWT_KEY`, `id` claim, expiry from `JWT_EXPIRATION`). `RequireAuth()` validates the token on every `/api/*` request and also accepts an already-loaded session, so API routes work from both external clients and logged-in browser sessions.

---

## Database layer

Raw SQL via **sqlx** — no ORM. This keeps every query transparent and profilable.

```
DATABASE_URL          → main read pool (25 max conns, 10 idle, 5 min lifetime)
WRITE_DATABASE_URL    → write pool  (10 max conns, 5 idle, 5 min lifetime)
```

Both pools fall back through `READ_DATABASE_URL` → `DATABASE_URL`, so a single-connection setup works with no extra config.

Each store wraps `*sqlx.DB` with typed query methods and is injected into handlers via the `Deps` struct — no globals:

| Store | Responsibilities |
|---|---|
| `AdminStore` | `FindByUsername`, `UpdateLastLogin` |
| `UserStore` | `ListUsers`, `GetUserByID`, `UpdatePassword`, `ToggleFavorite`, `ToggleSuspend`, `AddActivity`, `GetActivities`, `Restore`, … |
| `OrderStore` | `QueryPrintPerformance`, `QueryActiveUsersPerformance`, `QueryNewLabels`, `QueryLabelsByMarketplace`, `QueryLabelReportRows`, `QueryReturnLabelReportRows` |
| `LabelStore` | `GetByOrderID`, `UpdatePBStatus` |

---

## Data models

Defined in `internal/models/models.go`. Structs use `db:` tags for sqlx scanning and `json:` tags for the API layer. Nullable columns are modelled as pointers (`*string`, `*time.Time`).

| Model | Table | Notes |
|---|---|---|
| `Admin` | `admins` | Internal admin users |
| `User` | `users` | Customer accounts; `DeletedAt` for soft-delete |
| `UserRole` | `user_roles` | Admin / rep / regular roles |
| `Order` | `orders` | Joined with shipping services & statuses |
| `OrderLabel` | `order_labels` | Tracking ID, PDF URL, refund status |
| `Country` | `countries` | Active flag toggled from settings |
| `Marketplace` | `marketplaces` | Used in label report filtering |
| `AdminActivity` | `admin_activities` | Admin notes logged against a user |
| `PageResult[T]` | — | Generic pagination wrapper |

---

## Middleware

| Middleware | Role |
|---|---|
| `CORS()` | Origin whitelist from `CORS_ORIGIN` (exact, `regexp:` prefix, or `*`); `/ping` always allows `*` |
| `Sessions()` | Initializes the Gorilla cookie session store from `SESSION_SECRET` |
| `LoadSessionAdmin()` | Reads `admin_id` from the session into the Gin context (applied globally) |
| `SessionAuth()` | Gates protected HTML routes; redirects to `/login` when unauthenticated |
| `RequireAuth()` | Validates the JWT (or a loaded session) on `/api/*` routes |

---

## Views (templ + Datastar)

Templates live in `internal/views/`; each `.templ` compiles to a `*_templ.go` file via `make generate`.

- **`base.templ`** — the HTML shell (`Base`, `Dashboard`, `Sidebar`, `Topbar`), the Datastar CDN script, and the reusable lazy-load helpers (`LazySection`, `SkeletonCard/Stat/Table`, `ErrorNote`).
- **`pages/`** — one component per screen. Larger pages are split into a shell + independently-loading fragments (see below).

Datastar drives all interactivity with HTML attributes — no hand-written JavaScript. Debounced search (`data-on-input__debounce`), lazy section loading (`data-on-load`), and fragment swaps by matching element `id` are the core patterns; they're documented in depth in the tutorial (§13, §19).

---

## Performance: skeleton-first loading + parallel queries

Data-heavy pages render an **instant HTML shell** (zero DB work) with shimmering skeleton placeholders, then each section lazy-loads its own data:

- `LazySection(id, src)` renders a placeholder that fires `@get(src)` via Datastar's `data-on-load` the moment it enters the DOM.
- The fragment endpoint runs its query and returns just that section's HTML; Datastar swaps it in by matching the root element `id`.
- The dashboard, accounts/affiliates/USPS-reps/deactivated lists, and account/rep/affiliate detail pages all use this pattern.

Where a single endpoint needs several independent queries, they run **concurrently with goroutines**:

- `APIReportLabels` runs its main-service and return-label queries in parallel via `errgroup` — total latency is the slower query, not the sum.
- `APIGetLabel` fetches all label PDFs concurrently with a `sync.WaitGroup`, writing into a pre-sized indexed slice to preserve order without a mutex.

The connection pool (`SetMaxOpenConns(25)`) comfortably absorbs the concurrency. Full walkthrough with the data-race rules is in tutorial §19.

---

## Testing

- **Unit / build:** `make test` runs `go test ./...`.
- **End-to-end:** the tutorial (§20) includes a complete [Playwright](https://playwright.dev) e2e guide — config with an auto-boot `webServer`, login/redirect/auth-flow tests, and an assertion that proves the skeleton-first fragments actually swap in the browser (`not.toHaveClass(/animate-pulse/)`). Always point e2e tests at a throwaway database, never production.

---

## Known limitations

- **PDF merging (`APIGetLabel`)** concatenates PDF bytes naively — fine for single-page labels, invalid for multi-page. Swap in [pdfcpu](https://github.com/pdfcpu/pdfcpu) (pure Go) or [unipdf](https://github.com/unidoc/unipdf) before production use.
- **No read/write split wired in `main.go`** — `Connect()` uses `DATABASE_URL` for all stores. To add a read replica, call `store.WritableConnect()` too and pass separate pools to write-path stores.
- **`internal/services/` is a placeholder** — business logic currently lives inline in the handlers; extract it into service structs as the app grows.
