# Building a Full-Stack Go Admin Panel

## Go · Gin · templ · sqlx · Datastar · Tailwind CSS · air

This tutorial walks through the architecture of a server-rendered Go admin panel — a real, working application built with Gin, templ, Datastar, and Tailwind CSS.

It is written for **beginner to mid-level developers**. If you know what a web server does but haven't built one in Go, this tutorial will teach you the patterns. If you're coming from Node.js, Django, or Rails, you'll find direct comparisons throughout. Advanced readers will find deeper explanations in later sections.

---

## Table of contents

1. [The big picture: how a request becomes a web page](#1-the-big-picture-how-a-request-becomes-a-web-page)
2. [The stack and why](#2-the-stack-and-why)
3. [Project layout — why files live where they do](#3-project-layout--why-files-live-where-they-do)
4. [main.go — the application's front door](#4-maingo--the-applications-front-door)
5. [models — what your data looks like in Go](#5-models--what-your-data-looks-like-in-go)
6. [store — talking to the database](#6-store--talking-to-the-database)
7. [routes — the URL map](#7-routes--the-url-map)
8. [handlers — what happens when a URL is hit](#8-handlers--what-happens-when-a-url-is-hit)
9. [pages (templ) — turning data into HTML](#9-pages-templ--turning-data-into-html)
10. [layouts — the shared shell around every page](#10-layouts--the-shared-shell-around-every-page)
11. [middleware — code that runs before your handler](#11-middleware--code-that-runs-before-your-handler)
12. [Authentication: session cookies + JWT](#12-authentication-session-cookies--jwt)
13. [Reactive UI with Datastar](#13-reactive-ui-with-datastar)
14. [Styling with Tailwind CSS v3](#14-styling-with-tailwind-css-v3)
15. [Live reload with air](#15-live-reload-with-air)
16. [The build pipeline (Makefile)](#16-the-build-pipeline-makefile)
17. [End-to-end walkthrough: the accounts list](#17-end-to-end-walkthrough-the-accounts-list)
18. [Patterns and tradeoffs](#18-patterns-and-tradeoffs)
19. [Performance: skeleton-first pages + parallel queries](#19-performance-skeleton-first-pages--parallel-queries)
20. [End-to-end testing with Playwright](#20-end-to-end-testing-with-playwright)

---

## 1. The big picture: how a request becomes a web page

Before looking at any code, understand the flow. Every HTTP request the browser makes follows this path:

```
Browser sends: GET /accounts?q=john
        │
        ▼
   Gin router  (routes.go)
   "Which handler owns /accounts?"
        │
        ▼
   Middleware chain  (middleware/auth.go, session.go)
   "Is the user logged in? If not, redirect to /login."
        │
        ▼
   Handler function  (handlers/accounts_handler.go)
   "Read query params. Ask the store for data."
        │
        ▼
   Store  (store/user_store.go)
   "Run the SQL query. Return typed Go structs."
        │
        ▼
   Page component  (views/pages/accounts.templ)
   "Render the data as HTML."
        │
        ▼
Browser receives: full HTML page
```

Each box is a separate file (or small group of files). This separation is deliberate — each layer has **one job**, and you can change it without touching the others. A database query change stays in the store. A visual change stays in the templ file. A new URL stays in routes.go.

This is called **separation of concerns** and it's why codebases don't turn into spaghetti as they grow.

---

## 2. The stack and why

| Concern | Choice | What it replaces / comparison |
|---|---|---|
| HTTP routing | Gin | Like Express.js for Go — handles routing, params, middleware |
| HTML templates | templ | Like JSX but compiled to Go — type-safe, no runtime errors from bad data |
| Database | sqlx + raw SQL | Like `node-postgres` — write SQL yourself, auto-scans results into structs |
| Reactivity | Datastar | Like HTMX — HTML attributes drive fetch/update, no JavaScript written |
| CSS | Tailwind v3 | Same as in any React/Next.js project |
| Live reload | air | Like nodemon for Go |

The guiding principle: **no magic**. Every layer is transparent. The SQL queries are strings you can paste into a MySQL console. The HTML output is Go code you can read and debug. The "reactivity" is a 14 KB script tag — not a compiled JS bundle.

---

## 3. Project layout — why files live where they do

```
v2/
├── cmd/server/main.go         ← Start here — the entry point
├── internal/
│   ├── models/models.go       ← Data shapes (Go structs)
│   ├── store/                 ← Database queries — all SQL lives here
│   │   ├── db.go              ← Connect to MySQL
│   │   ├── user_store.go      ← Queries about users
│   │   ├── order_store.go     ← Queries about orders/labels
│   │   ├── admin_store.go     ← Queries about admins
│   │   └── label_store.go     ← Queries about labels
│   ├── http/handlers/         ← Handle HTTP requests
│   │   ├── deps.go            ← Shared dependencies (stores)
│   │   ├── routes.go          ← URL → handler mapping
│   │   ├── auth_handler.go    ← Login, logout
│   │   ├── accounts_handler.go
│   │   ├── home_handler.go
│   │   ├── labels_handler.go
│   │   ├── affiliates_handler.go
│   │   └── settings_handler.go
│   ├── middleware/            ← Code that runs before handlers
│   │   ├── auth.go            ← JWT token checking
│   │   ├── cors.go            ← Cross-origin request rules
│   │   └── session.go         ← Cookie session management
│   └── views/
│       ├── layouts/base.templ ← The HTML wrapper every page uses
│       └── pages/             ← One .templ file per page
│           ├── login.templ
│           ├── accounts.templ
│           ├── account_detail.templ
│           └── ...
├── static/css/
│   ├── input.css              ← Tailwind source (you edit this)
│   └── app.css                ← Tailwind output (generated, don't edit)
├── .air.toml                  ← Live reload config
├── Makefile                   ← Build commands
└── tailwind.config.js
```

### Why `internal/`?

Go has a built-in rule: packages inside an `internal/` directory can only be imported by code in the **same module**. This means no other Go project can accidentally import your `store` or `models` packages. It's a compiler-enforced boundary that keeps the codebase from becoming entangled with outside code.

### Why `cmd/server/`?

The `cmd/` directory is a Go convention for executable entry points. If you later add a second binary — a CLI tool or a background worker — it gets its own `cmd/worker/main.go`. Each sub-directory under `cmd/` produces one binary. Shared business logic stays in `internal/` and is reused by all of them.

### Why split handlers into multiple files?

All files in `handlers/` belong to the same `handlers` package. Splitting by domain (accounts, affiliates, labels) keeps each file focused on one area. When you're working on account logic, you open `accounts_handler.go` — you don't wade through 800 lines of unrelated code.

---

## 4. main.go — the application's front door

`main.go` is the only file that directly touches everything. It's the glue. Its job is to:

1. Load configuration (environment variables)
2. Connect to the database
3. Build all the stores
4. Build the handler (injecting the stores into it)
5. Set up the router and its middleware
6. Start the server

```go
// cmd/server/main.go
func main() {
    // 1. Load .env file into environment variables.
    //    If there's no .env file, reads from the process environment (correct for production).
    godotenv.Load()

    // 2. Connect to MySQL
    db, err := store.Connect()
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // 3. Build stores — each store wraps the DB connection
    //    and provides typed query methods.
    adminStore := store.NewAdminStore(db)
    userStore  := store.NewUserStore(db)
    orderStore := store.NewOrderStore(db)
    labelStore := store.NewLabelStore(db)

    // 4. Build the handler — pass all stores in so handlers can use them.
    //    This is called dependency injection.
    h := handlers.New(handlers.Deps{
        AdminStore: adminStore,
        UserStore:  userStore,
        OrderStore: orderStore,
        LabelStore: labelStore,
    })

    // 5. Create the Gin router
    r := gin.Default()              // includes request logger + crash recovery
    r.SetTrustedProxies(nil)        // don't trust X-Forwarded-For (no load balancer locally)
    r.Static("/static", "./static") // serve CSS/images from the static/ directory
    r.Use(middleware.CORS())        // apply CORS headers to every response

    h.RegisterRoutes(r)

    // 6. Start the HTTP server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    r.Run(":" + port)
}
```

### Why is main.go so short?

Intentionally. `main.go` does construction, not logic. It creates things and wires them together. All actual business logic lives in `internal/`. This makes it easy to read and the dependencies are explicit — you can see at a glance what the application needs to start.

### What is dependency injection?

When `handlers.New(handlers.Deps{...})` is called, the stores are **injected** into the handler. The handler doesn't create its own database connection — it receives one from the caller. This is dependency injection (DI).

Why does it matter?
- In tests, you can pass a fake store instead of a real database
- In `main.go`, you can see exactly what each part of the app needs to function
- No hidden global state that causes surprises

### `gin.Default()` vs `gin.New()`

`gin.Default()` includes two built-in middleware automatically:
- **Logger** — prints every request to stdout: `GET /accounts 200 1.2ms`
- **Recovery** — catches panics and returns a 500 error instead of crashing the server

`gin.New()` starts completely bare. Use `gin.Default()` for development; in production swap the logger for a structured one (e.g. `zap`).

---

## 5. models — what your data looks like in Go

Models are plain Go structs that mirror your database tables. No methods, no active record pattern — just data containers. They live in `internal/models/models.go`.

```go
type User struct {
    ID        uint    `db:"id"         json:"id"`
    Name      string  `db:"name"       json:"name"`
    Email     string  `db:"email"      json:"email"`
    Company   *string `db:"company"    json:"company"`   // nullable → pointer
    Suspend   bool    `db:"suspend"    json:"suspend"`
    Favorite  bool    `db:"favorite"   json:"favorite"`
    DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"` // nil = not deleted
    CreatedAt *time.Time `db:"created_at" json:"created_at"`
}
```

### What are struct tags?

The backtick annotations after each field are **struct tags** — metadata that tells external libraries how to interpret this field:

- `db:"column_name"` — used by **sqlx** to map a database column to this field when scanning query results. Without it, sqlx would try to match `CreatedAt` to a column named `CreatedAt` (capital C), but the database column is `created_at` (lowercase).
- `json:"key_name"` — used when serializing this struct to JSON for the API. Without it, the JSON key would be `CreatedAt` instead of `created_at`.

### Why pointers for nullable fields?

In Go, `string` always has a value — it can't be absent. But in the database, `company` might be `NULL`. A `*string` can be either `nil` (NULL) or point to an actual string. This lets you tell the difference between "empty string" and "not set."

```go
Company *string  // nil means NULL in the database
```

In templates, handle this with a helper:

```go
func strOrDash(s *string) string {
    if s == nil { return "—" }
    return *s  // *s dereferences the pointer to get the string value
}
```

### Models are passive

Models don't have methods that talk to the database (unlike Active Record in Rails or Django ORM). All database logic lives in the store layer. This keeps models simple and testable in isolation.

---

## 6. store — talking to the database

The store layer is where all SQL lives. Nothing outside this directory writes SQL. When a query is slow or wrong, you look here.

### The database connection

```go
// internal/store/db.go
func Connect() (*sqlx.DB, error) {
    dsn := os.Getenv("DATABASE_URL")
    // Format: user:password@tcp(host:port)/dbname?parseTime=true
    // parseTime=true tells the MySQL driver to return DATETIME columns as time.Time

    db, err := sqlx.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    // Connection pool — reuse connections across requests
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(5 * time.Minute)

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("unable to reach database: %w", err)
    }
    return db, nil
}
```

`sqlx.Open` is **lazy** — it doesn't connect until you call `Ping()`. The pool means the app doesn't open a new database connection for every HTTP request.

### Store structs

Each store wraps `*sqlx.DB` with typed query methods:

```go
type UserStore struct {
    db *sqlx.DB
}

func NewUserStore(db *sqlx.DB) *UserStore {
    return &UserStore{db: db}
}

func (s *UserStore) GetUserByID(id uint) (*models.User, error) {
    var u models.User
    err := s.db.Get(&u, `SELECT id, name, email, company, suspend, favorite,
                                deleted_at, created_at
                         FROM users WHERE id = ?`, id)
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

### `sqlx.Get` vs `sqlx.Select`

```go
// Expecting one row — use Get
var u models.User
err := s.db.Get(&u, "SELECT ... WHERE id = ?", id)
// If no row found: err == sql.ErrNoRows

// Expecting many rows — use Select
var users []models.User
err := s.db.Select(&users, "SELECT ... WHERE role_id = ?", roleID)
// If no rows found: users is an empty slice, err is nil
```

### Dynamic query building

When a query has optional conditions, build the SQL string programmatically with a filter type:

```go
type UserFilter struct {
    Search  string
    RoleID  *uint  // nil = don't filter by role
    Deleted bool   // true = show only soft-deleted users
    Page    int
    Limit   int
}

func buildUserWhere(f UserFilter) (string, []interface{}) {
    var conds []string
    var args  []interface{}

    if f.Deleted {
        conds = append(conds, "u.deleted_at IS NOT NULL")
    } else {
        conds = append(conds, "u.deleted_at IS NULL")
    }

    if f.RoleID != nil {
        conds = append(conds, "u.user_role_id = ?")
        args = append(args, *f.RoleID)
    }

    if f.Search != "" {
        s := "%" + f.Search + "%"  // SQL LIKE wildcard
        conds = append(conds, "(u.name LIKE ? OR u.email LIKE ?)")
        args = append(args, s, s)
    }

    if len(conds) == 0 {
        return "", args
    }
    return "WHERE " + strings.Join(conds, " AND "), args
}
```

SQL conditions and their argument values are **built together** — each `?` placeholder and its matching value in `args` are appended as a pair, so they never go out of sync.

### Pagination

```go
func (s *UserStore) ListUsers(f UserFilter) ([]models.User, int64, error) {
    where, args := buildUserWhere(f)
    offset := (f.Page - 1) * f.Limit  // page 1 = offset 0, page 2 = offset 50

    // Count query — for "showing X of Y" display
    var total int64
    s.db.Get(&total,
        "SELECT COUNT(*) FROM users u LEFT JOIN user_roles ur ON ur.id = u.user_role_id "+where,
        args...)

    // Data query — add LIMIT and OFFSET at the end
    listArgs := append(args, f.Limit, offset)
    var users []models.User
    s.db.Select(&users,
        "SELECT ... FROM users u ... "+where+" ORDER BY u.created_at DESC LIMIT ? OFFSET ?",
        listArgs...)

    return users, total, nil
}
```

Two queries share the same `WHERE` clause — the total count and the page of data always stay in sync.

---

## 7. routes — the URL map

`routes.go` maps every URL to a handler function. Think of it as the table of contents for the entire application.

```go
// internal/http/handlers/routes.go
func (h *Handler) RegisterRoutes(r *gin.Engine) {

    // Health check — no auth required
    r.GET("/ping", h.Ping)

    // Session middleware applied globally — must come before any route that reads the session
    r.Use(middleware.Sessions())
    r.Use(middleware.LoadSessionAdmin())

    // ── Public HTML routes ────────────────────────────────────────────────
    r.GET("/login",  h.GetLogin)
    r.POST("/login", h.PostLogin)
    r.GET("/logout", h.Logout)

    // ── Protected HTML routes — SessionAuth() runs first ──────────────────
    protected := r.Group("/", middleware.SessionAuth())
    {
        protected.GET("/",                               h.GetHome)
        protected.GET("/accounts",                       h.GetAccounts)
        protected.GET("/accounts/:id",                   h.GetAccountDetail)
        protected.POST("/accounts/:id/change-password",  h.PostAccountChangePassword)
        protected.POST("/accounts/:id/favorite",         h.PostToggleFavorite)
        protected.POST("/accounts/:id/suspend",          h.PostToggleSuspend)
        protected.POST("/accounts/:id/activity",         h.PostAddActivity)
        // ...
    }

    // ── JSON API routes — RequireAuth() checks JWT header ─────────────────
    api := r.Group("/api", middleware.RequireAuth())
    {
        api.POST("/login",                       h.APILogin)
        api.POST("/reports/print-performance",   h.APIReportPrintPerformance)
        api.GET("/get-total-billed/:customerId", h.APIGetTotalBilled)
        // ...
    }
}
```

### Why keep all routes in one file?

When something breaks, the first question is "which handler owns this URL?" With all routes in one file, the answer is one `Ctrl+F` away. You never hunt through handler files.

### Route groups: `r.Group()`

A route group applies middleware to a set of routes at once:

```go
// Without groups — repeat middleware on every route:
r.GET("/accounts", middleware.SessionAuth(), h.GetAccounts)
r.GET("/accounts/:id", middleware.SessionAuth(), h.GetAccountDetail)

// With a group — apply once:
protected := r.Group("/", middleware.SessionAuth())
protected.GET("/accounts", h.GetAccounts)
protected.GET("/accounts/:id", h.GetAccountDetail)
```

### URL parameters: `:id`

A colon prefix creates a named URL parameter:

```go
protected.GET("/accounts/:id", h.GetAccountDetail)
```

Read it in the handler with `c.Param("id")`:

```go
idStr := c.Param("id")                       // "42"
id, _ := strconv.ParseUint(idStr, 10, 64)   // convert to number
```

### Two route groups for two audiences

This app serves:
- **Browser users** — admins logging in via the HTML UI. They get a session cookie after login; the browser sends it automatically on every request.
- **API clients** — a separate frontend app, mobile client, or integration. They send a JWT token in a header on every request.

Two route groups, two auth strategies, one binary.

---

## 8. handlers — what happens when a URL is hit

A handler is a Go function that receives an HTTP request and writes a response. In Gin, every handler has this signature:

```go
func(c *gin.Context)
```

`*gin.Context` gives you everything about the request (query params, form data, URL params, headers) and the tools to write a response (HTML, JSON, redirects).

### The `Handler` struct and `Deps`

All handler functions are methods on a single `Handler` struct, which holds the stores they need:

```go
// internal/http/handlers/deps.go
type Deps struct {
    AdminStore *store.AdminStore
    UserStore  *store.UserStore
    OrderStore *store.OrderStore
    LabelStore *store.LabelStore
}

type Handler struct {
    Deps  // embedding promotes Deps fields — access as h.UserStore, not h.Deps.UserStore
}

func New(d Deps) *Handler {
    return &Handler{Deps: d}
}
```

### An HTML handler

```go
func (h *Handler) GetAccountDetail(c *gin.Context) {
    // 1. Parse the :id URL parameter
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 64)
    if err != nil {
        c.Status(http.StatusBadRequest)
        return
    }

    // 2. Fetch data from the store
    u, err := h.UserStore.GetUserByID(uint(id))
    if err != nil {
        c.Status(http.StatusNotFound)
        return
    }
    activities, _ := h.UserStore.GetActivities(u.ID)
    flash := c.Query("flash")  // optional ?flash=Password+updated

    // 3. Render the templ component and stream it as HTML
    c.Header("Content-Type", "text/html; charset=utf-8")
    pages.AccountDetail(pages.AccountDetailData{
        User:       u,
        Activities: activities,
        Flash:      flash,
    }).Render(c.Request.Context(), c.Writer)
}
```

The handler's job: parse inputs → get data → render output. No SQL. No HTML building.

### A JSON API handler

```go
func (h *Handler) APILogin(c *gin.Context) {
    var req struct {
        Email    string `json:"email"    binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    admin, err := h.AdminStore.FindByUsername(req.Email)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Email or password don't match"})
        return
    }
    if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Email or password don't match"})
        return
    }

    token, _ := generateJWT(admin.ID)
    c.JSON(http.StatusOK, token)
}
```

### The `loadUser` shared helper

Several handlers parse `:id` and look up the user. Rather than repeat it everywhere, extract a helper:

```go
func loadUser(c *gin.Context, h *Handler) (*models.User, bool) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        c.Status(http.StatusBadRequest)
        return nil, false
    }
    u, err := h.UserStore.GetUserByID(uint(id))
    if err != nil {
        c.Status(http.StatusNotFound)
        return nil, false
    }
    return u, true
}
```

Handlers call it like this:

```go
func (h *Handler) PostToggleFavorite(c *gin.Context) {
    u, ok := loadUser(c, h)
    if !ok { return }  // response already written — just stop

    fav := c.PostForm("favorite") == "true"
    h.UserStore.ToggleFavorite(u.ID, fav)
    c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Updated", u.ID))
}
```

### Flash messages

After a successful action, redirect with `?flash=` in the URL:

```go
c.Redirect(http.StatusFound, fmt.Sprintf("/accounts/%d?flash=Password+updated", u.ID))
```

The destination handler reads it and passes it to the template, which shows it if non-empty. The message disappears on the next refresh — correct behavior for a one-time success notice.

---

## 9. pages (templ) — turning data into HTML

[templ](https://templ.guide) is a Go library that compiles `.templ` files to type-safe Go code. Think of it like JSX — HTML mixed with Go logic — but it runs on the server and produces plain HTML.

### Why templ instead of Go's built-in `html/template`?

Go's standard library `html/template` uses string-based templates where errors only appear at runtime. templ compiles to Go, so missing variables or wrong types are **compile-time errors** — you find them before users do.

### Basic syntax

```go
// internal/views/pages/login.templ
package pages

templ Login(errorMsg string) {
    @layouts.Base("Login") {
        <div class="min-h-screen flex items-center justify-center">

            if errorMsg != "" {
                <p class="text-red-600 text-sm">{ errorMsg }</p>
            }

            <form method="post" action="/login">
                <input type="text" name="email" placeholder="Email"/>
                <input type="password" name="password" placeholder="Password"/>
                <button type="submit">Sign In</button>
            </form>

        </div>
    }
}
```

Key syntax rules:
- `{ variable }` — renders a string. **HTML-escaped automatically** — no XSS.
- `if condition { }` — standard Go `if`, no extra template syntax
- `for _, item := range items { }` — standard Go `for`
- `@ComponentName(args)` — renders another templ component inline
- `{ children... }` — slot: renders whatever the caller passes in `{ }`

### Passing data to a component

Define a data struct for each page:

```go
type AccountDetailData struct {
    User       *models.User
    Activities []models.AdminActivity
    Flash      string
}

templ AccountDetail(d AccountDetailData) {
    @layouts.Dashboard(fmt.Sprintf("Account: %s", d.User.Name)) {
        if d.Flash != "" {
            <div class="bg-green-50 text-green-700">{ d.Flash }</div>
        }
        <h1>{ d.User.Name }</h1>
        <p>{ d.User.Email }</p>
    }
}
```

Using a struct means the compiler tells you if you forgot a field or passed the wrong type.

### Rendering in a handler

```go
c.Header("Content-Type", "text/html; charset=utf-8")
pages.AccountDetail(pages.AccountDetailData{
    User:       u,
    Activities: activities,
    Flash:      flash,
}).Render(c.Request.Context(), c.Writer)
```

`Render(ctx, w)` writes HTML directly to the HTTP response writer. No buffering, no intermediate string — it streams as it generates.

### Helper functions

Regular Go functions can live alongside components in a `.templ` file:

```go
func fmtTime(t *time.Time) string {
    if t == nil { return "—" }
    return t.Format("Jan 2, 2006")
}

func strOrDash(s *string) string {
    if s == nil { return "—" }
    return *s
}
```

Use them directly in templates:

```go
<td>{ fmtTime(u.CreatedAt) }</td>
<td>{ strOrDash(u.Company) }</td>
```

### Sub-components

Large pages are split into smaller private components. Lowercase names are unexported (private to the package):

```go
templ AccountDetail(d AccountDetailData) {
    @layouts.Dashboard("...") {
        <div class="grid grid-cols-3 gap-6">
            @userInfoCard(d.User)                     // private — only used here
            @activitiesCard(d.Activities, d.User.ID)  // private
        </div>
    }
}

// Only callable within this package
templ userInfoCard(u *models.User) {
    <div class="bg-white rounded-xl border">
        ...
    </div>
}
```

### Safe URLs

templ prevents URL injection. Dynamic values in `href` attributes must go through `templ.URL()`:

```go
// ✅ Safe — templ.URL validates and escapes the value
<a href={ templ.URL(fmt.Sprintf("/accounts/%d", u.ID)) }>View</a>

// ❌ Won't compile — templ rejects raw expressions in href
<a href={ "/accounts/" + someVar }>View</a>
```

This prevents attacker-controlled strings from becoming `javascript:evil()` in an href.

### Generated files

`make generate` (or `templ generate ./...`) writes a `*_templ.go` file next to each `.templ` file. These are regular Go code you can read and debug, but they're overwritten on every `make generate` — don't edit them.

---

## 10. layouts — the shared shell around every page

A layout is a component that wraps other components. It provides the HTML skeleton, CSS link, scripts, and navigation that every page shares.

```go
// internal/views/layouts/base.templ

// Base: outermost HTML shell
templ Base(title string) {
    <!DOCTYPE html>
    <html lang="en" class="h-full">
    <head>
        <meta charset="UTF-8"/>
        <title>{ title } — GrandShipper Admin</title>
        <link rel="stylesheet" href="/static/css/app.css"/>
        <script type="module"
            src="https://cdn.jsdelivr.net/npm/@sudodevnull/datastar@0.20.1/dist/datastar.iife.js">
        </script>
    </head>
    <body class="h-full bg-gray-50">
        { children... }    ← whatever the caller puts in { } goes here
    </body>
    </html>
}

// Dashboard: adds sidebar + topbar around the page content
templ Dashboard(title string) {
    @Base(title) {
        <div class="flex h-full">
            @Sidebar()
            <div class="flex flex-col flex-1">
                @Topbar(title)
                <main class="flex-1 overflow-y-auto p-6">
                    { children... }   ← page content lands here
                </main>
            </div>
        </div>
    }
}
```

### How nesting works

When a page calls `@layouts.Dashboard("Accounts") { ... }`, its content fills the `{ children... }` slot inside Dashboard. Dashboard in turn calls `@Base(title) { ... }`, so its own content fills the slot inside Base. The chain:

```
Page content
  → Dashboard (sidebar + topbar, wrapped in Base)
      → Base (HTML boilerplate, CSS, Datastar script)
          → Full HTML response sent to browser
```

No "extends" keyword. No inheritance. Just components calling components, each passing their slot content down.

### Why separate `Base` and `Dashboard`?

The login page doesn't have a sidebar — it uses `@layouts.Base("Login")` directly. The dashboard pages use `@layouts.Dashboard("Accounts")`. Both share the HTML boilerplate from `Base`. Separating them avoids special-casing the login page inside the layout.

---

## 11. middleware — code that runs before your handler

Middleware is a function that runs before (and optionally after) your handler for every matching request. It can add data to the context, check auth, or abort the request (e.g. "not logged in → redirect to /login").

In Gin, middleware has the same signature as a handler — `func(c *gin.Context)` — but it calls `c.Next()` to pass control forward.

```
Request
   │
   ▼
Sessions()         ← reads/writes the session cookie
   │
   ▼
LoadSessionAdmin() ← reads admin ID from session into the Gin context
   │
   ▼
SessionAuth()      ← checks if admin ID is in context; redirects if not
   │
   ▼
YourHandler()      ← finally, the actual handler runs
```

### How `c.Next()` and `c.Abort()` work

```go
func MyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Code here runs BEFORE the handler

        if notAuthorized {
            c.Redirect(http.StatusFound, "/login")
            c.Abort()   // stop the chain — handler will NOT run
            return
        }

        c.Next()  // pass control to the next middleware or handler

        // Code here runs AFTER the handler (only if Abort wasn't called)
    }
}
```

Without `c.Abort()`, the chain continues even after a redirect — meaning the handler runs and potentially writes a second response.

### Session middleware (three functions, three jobs)

```go
// 1. Initialize the session store (must run first on every request)
func Sessions() gin.HandlerFunc {
    secret := os.Getenv("SESSION_SECRET")
    store := cookie.NewStore([]byte(secret))
    return ginsessions.Sessions("admin_session", store)
}

// 2. Read admin ID from the session cookie into Gin's context
//    (applied globally — even public routes can see if someone is logged in)
func LoadSessionAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := ginsessions.Default(c)
        v := session.Get("admin_id")
        if v != nil {
            c.Set("admin_id", v)
        }
        c.Next()
    }
}

// 3. Gate access — redirect to /login if no admin_id in context
//    (applied only to protected routes via r.Group)
func SessionAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        if _, exists := c.Get("admin_id"); !exists {
            c.Redirect(http.StatusFound, "/login")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

Why three separate functions? `LoadSessionAdmin` runs on every request — including public ones — so templates can optionally show the logged-in admin's name. `SessionAuth` runs only on protected routes. Separating them keeps each function's responsibility clear.

---

## 12. Authentication: session cookies + JWT

The app has two authentication systems running side by side, because it serves two different types of clients.

### Why two systems?

- **Browser users** (admins) log in once through a form. The browser automatically sends a cookie on every subsequent request. This is the session cookie system.
- **API clients** (a separate frontend, mobile app, or integration) send a JWT token in a header on every request. Cookies don't work well for programmatic clients.

Both systems set the same `admin_id` value in Gin's context. Once it's there, the rest of the code doesn't need to know how the user authenticated.

### Session cookies — for the browser

The session store uses Gorilla's encrypted cookies. No database, no Redis — the session data is encrypted and signed inside the cookie itself, and only the server (which holds `SESSION_SECRET`) can read or forge it.

```go
// internal/middleware/session.go
func Sessions() gin.HandlerFunc {
    secret := os.Getenv("SESSION_SECRET") // 32-byte random string
    store := cookie.NewStore([]byte(secret))
    return ginsessions.Sessions("admin_session", store)
}

func SetSessionAdmin(c *gin.Context, adminID uint) {
    session := ginsessions.Default(c)
    session.Set("admin_id", adminID)
    session.Save()
}
```

The login handler verifies the password, then writes the admin's ID into the session:

```go
func (h *Handler) PostLogin(c *gin.Context) {
    email    := c.PostForm("email")
    password := c.PostForm("password")

    admin, err := h.AdminStore.FindByUsername(email)
    if err != nil {
        // Same error for "user not found" and "wrong password" —
        // prevents attackers from discovering which emails are registered
        renderLoginError(c, "Email or password don't match")
        return
    }

    // bcrypt comparison runs in constant time — safe against timing attacks
    if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
        renderLoginError(c, "Email or password don't match")
        return
    }

    h.AdminStore.UpdateLastLogin(admin.ID)
    middleware.SetSessionAdmin(c, admin.ID)  // write session cookie
    c.Redirect(http.StatusFound, "/")
}
```

Why is the error message identical for "user not found" and "wrong password"? If it said "no such user" for one and "wrong password" for the other, an attacker could probe which email addresses have accounts. Returning the same message for both closes that leak.

### JWT — for API clients

A JWT (JSON Web Token) is a signed token the client stores and sends on every request. The server can verify it without looking anything up in a database — the signature proves the token is authentic.

```go
func generateJWT(adminID uint) (string, error) {
    secret := []byte(os.Getenv("JWT_KEY"))
    claims := jwt.MapClaims{
        "id":  adminID,
        "exp": time.Now().Add(8 * time.Hour).Unix(),  // expires in 8 hours
    }
    return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}
```

The API login endpoint returns this token:

```go
func (h *Handler) APILogin(c *gin.Context) {
    // ... validate credentials ...
    token, _ := generateJWT(admin.ID)
    c.JSON(http.StatusOK, token)
}
```

API clients then send it on every request as either:
```
x-auth-token: eyJhbGci...
```
or:
```
Authorization: Bearer eyJhbGci...
```

The `RequireAuth` middleware verifies it. It accepts either header (for compatibility with the existing frontend) and also falls back to the session — so the same API routes work from both a logged-in browser and an external client:

```go
func RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("x-auth-token")
        if token == "" {
            token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        }
        // Fall back to session auth if no token (hybrid routes)
        if token == "" {
            if _, exists := c.Get("admin_id"); exists {
                c.Next()
                return
            }
        }
        claims, err := parseJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        c.Set("admin_id", claims)
        c.Next()
    }
}
```

---

## 13. Reactive UI with Datastar

[Datastar](https://data-star.dev) is a hypermedia framework that wires reactive signals to SSE-driven DOM updates using only HTML attributes — no build step, no virtual DOM, no JavaScript you write. It's loaded from CDN as a single `<script>` tag (~14 KB, ESM module):

```html
<!-- layouts/base.templ -->
<script type="module"
    src="https://cdn.jsdelivr.net/npm/@sudodevnull/datastar@0.20.1/dist/datastar.iife.js">
</script>
```

Datastar is deliberately different from HTMX. The key distinction: Datastar has a built-in signal graph (`data-store`/`data-bind`), so client state lives as declared reactive values rather than being encoded in URL parameters or form fields. This matters for debounced search, multi-field filter forms, and any UI where several inputs jointly determine what gets fetched.

---

### Core concepts

#### Signals: `data-store` + `data-bind`

`data-store` declares a reactive signal map on any element. Signals are scoped to the element and all its descendants (like CSS custom properties).

```html
<!-- Declare signals -->
<div data-store='{"q": "", "page": 1, "loading": false}'>

    <!-- Two-way bind: input value ↔ $q signal -->
    <input type="text" data-bind="q" placeholder="Search…"/>

    <!-- One-way read: show the current value of $page -->
    <span data-text="$page"></span>

    <!-- Conditional visibility: show spinner while $loading is true -->
    <div data-show="$loading">Loading…</div>

</div>
```

Key attributes:
- `data-store='{"key": value}'` — declares signals. Values are JSON literals.
- `data-bind="signalName"` — two-way binds an input/select/textarea to a signal.
- `data-text="$signal"` — sets the element's `textContent` reactively.
- `data-show="$signal"` — toggles `display:none` based on a boolean expression.
- `data-class-foo="$signal"` — conditionally applies class `foo`.

Signals are referenced as `$signalName` in expressions. Signal scope is hierarchical: a child element can read signals declared on any ancestor.

#### Event reactions: `data-on-*`

`data-on-<event>` attaches a handler to any DOM event. The value is a Datastar expression, not arbitrary JavaScript.

```html
<!-- Fire GET request on every click -->
<button data-on-click="@get('/api/ping')">Ping</button>

<!-- Fire GET on input, debounced 300ms, using current signal value -->
<input data-bind="q" data-on-input__debounce.300ms="@get('/accounts/search?q=' + $q)"/>

<!-- Fire POST on form submit, prevent native browser POST -->
<form data-on-submit__prevent="@post('/accounts/create')">...</form>

<!-- Run on page load (fires once when the element enters the DOM) -->
<div data-on-load="@get('/stats/summary')"></div>
```

**Modifiers** are appended with `__`:
- `__debounce.300ms` — wait N ms of silence before firing (resets on each event)
- `__throttle.500ms` — fire at most once per N ms
- `__prevent` — call `event.preventDefault()`
- `__stop` — call `event.stopPropagation()`
- `__once` — fire only once, then remove the listener

You can chain modifiers: `data-on-input__debounce.300ms__prevent="..."`.

#### Actions: `@get`, `@post`, `@patch`, `@put`, `@delete`

These are Datastar's HTTP actions. They fire a fetch request to the given URL and merge the response HTML into the DOM.

```html
data-on-click="@get('/accounts/search?q=' + $q)"
data-on-submit__prevent="@post('/accounts/' + $id + '/favorite')"
```

The response is expected to contain one or more HTML fragments. Datastar finds each fragment's root element by `id` and replaces the matching element in the live DOM. If no `id` matches, the response is ignored (logged in dev mode).

**Merging strategy**: by default Datastar replaces the `innerHTML` of the matching element. You can control this with `data-merge-mode` on the returned fragment element:
- `data-merge-mode="morph"` — morphdom diff, preserves input focus
- `data-merge-mode="inner"` — replace innerHTML (default)
- `data-merge-mode="outer"` — replace outerHTML (replaces the element itself)
- `data-merge-mode="prepend"` / `data-merge-mode="append"` — insert before/after existing children

#### SSE responses (advanced)

For streaming updates, Datastar supports Server-Sent Events. Instead of returning plain HTML, the server writes an SSE stream with `Content-Type: text/event-stream`. Each event can patch signals or merge HTML fragments:

```
event: datastar-merge-fragments
data: fragments <div id="stats">...</div>

event: datastar-merge-signals
data: signals {"loading":false,"count":42}
```

This is useful for long-running operations (report generation, file uploads) where you want to stream progress without polling.

In Gin:

```go
func (h *Handler) StreamReport(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    flusher, _ := c.Writer.(http.Flusher)

    // Send a signal update
    fmt.Fprintf(c.Writer, "event: datastar-merge-signals\ndata: signals {\"loading\":true}\n\n")
    flusher.Flush()

    // Do the work...
    rows := computeReport()

    // Send the HTML fragment
    var buf strings.Builder
    pages.ReportTable(rows).Render(c.Request.Context(), &buf)
    fmt.Fprintf(c.Writer, "event: datastar-merge-fragments\ndata: fragments %s\n\n", buf.String())

    // Signal done
    fmt.Fprintf(c.Writer, "event: datastar-merge-signals\ndata: signals {\"loading\":false}\n\n")
    flusher.Flush()
}
```

---

### The fragment pattern

This is the most important pattern in a Datastar + templ app. Every page that has a refreshable section is split into two components:

1. **The full page** — rendered on first load, contains the layout and the Datastar wiring
2. **The inner fragment** — rendered on partial updates, contains only the replaceable section

```go
// Full page — served at GET /accounts
templ Accounts(d AccountsData) {
    @layouts.Dashboard("Accounts") {
        // Search form with Datastar signals
        <form method="get" action="/accounts"
            data-store='{"q": ""}'
            data-on-input__debounce.300ms="@get('/accounts/search?q=' + $q)">
            <input
                type="text"
                name="q"
                value={ d.Search }
                data-bind="q"
                placeholder="Search accounts…"
            />
            <button type="submit">Search</button>
        </form>

        // This div is replaced on every search update
        <div id="accounts-table">
            @AccountsTable(d)
        </div>
    }
}

// Inner fragment — served at GET /accounts/search
// Returns ONLY this div; Datastar replaces #accounts-table in the DOM.
templ AccountsTable(d AccountsData) {
    <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
            <thead>
                <tr>
                    <th>Name</th><th>Email</th><th>Status</th>
                </tr>
            </thead>
            <tbody>
                for _, u := range d.Users {
                    <tr>
                        <td>{ u.Name }</td>
                        <td>{ u.Email }</td>
                        <td>
                            if u.Suspend {
                                <span class="badge-red">Suspended</span>
                            } else {
                                <span class="badge-green">Active</span>
                            }
                        </td>
                    </tr>
                }
            </tbody>
        </table>
    </div>
    @Pagination("/accounts", d.Page, d.Limit, d.TotalItems, d.Search)
}
```

The handler for the fragment endpoint:

```go
// GET /accounts/search — returns only the AccountsTable fragment
func (h *Handler) GetAccountsSearch(c *gin.Context) {
    f := buildUserFilter(c, nil)
    users, total, _ := h.UserStore.ListUsers(f)

    c.Header("Content-Type", "text/html; charset=utf-8")
    pages.AccountsTable(pages.AccountsData{
        Users: users, TotalItems: total,
        Page: f.Page, Limit: f.Limit, Search: f.Search,
    }).Render(c.Request.Context(), c.Writer)
}
```

Datastar receives the HTML, finds the root element's `id="accounts-table"` (or the first element if no match), and replaces the live DOM node. No full navigation, no scroll reset, no lost input focus elsewhere on the page.

---

### Inline form actions (toggles, soft actions)

When a form action should update part of the page without a full redirect:

```go
// The toggle button renders as a form for progressive enhancement
templ accountActionsCard(u *models.User) {
    <form method="post" action={ templ.URL(fmt.Sprintf("/accounts/%d/favorite", u.ID)) }
        data-on-submit__prevent="@post('/accounts/' + $id + '/favorite')">
        <input type="hidden" name="favorite" value={ boolToggleStr(!u.Favorite) }/>
        <input type="hidden" data-bind-id="id" value={ fmt.Sprintf("%d", u.ID) }/>
        <button type="submit">
            if u.Favorite { Remove Favorite } else { Add to Favorites }
        </button>
    </form>
}
```

The server handler responds with a fragment for the actions card, which gets swapped in. If JavaScript is disabled, the native form POST fires instead — full-page reload, same result.

---

### When to use Datastar vs. a plain form

| Scenario | Datastar | Plain form |
|---|---|---|
| Search with debounce | ✅ `data-on-input__debounce` | ❌ page reloads on every keystroke |
| Inline toggle (favorite, suspend) | ✅ swap action card fragment | ✅ acceptable redirect |
| Password change with feedback | ✅ `data-on-submit__prevent` + fragment | ✅ redirect with flash query param |
| Full form creation (new affiliate) | ✅ or ❌ — no clear win | ✅ simpler, no extra route |
| Real-time streaming (report progress) | ✅ SSE stream | ❌ not possible |
| Static page (settings list) | ❌ no benefit | ✅ simpler |

Use Datastar where a full-page reload creates noticeable UX friction. Static pages that redirect after a POST need nothing from Datastar.

---

### Datastar vs. HTMX (practical differences)

| Feature | Datastar | HTMX |
|---|---|---|
| Signal graph | ✅ `data-store` / `data-bind` | ❌ no built-in state |
| SSE streaming | ✅ native, structured | ✅ `hx-ext="sse"` |
| Debounce modifier | ✅ `__debounce.300ms` | ✅ `hx-trigger="keyup delay:300ms"` |
| Merge modes | ✅ `data-merge-mode` | ✅ `hx-swap` |
| Client-side expressions | ✅ `$signal` in attributes | ❌ no expression language |
| Bundle size | ~14 KB | ~14 KB |
| Ecosystem maturity | younger | older, more resources |

The signal graph is the decisive difference for this app. The search form needs to compose `$q` + `$page` + `$roleFilter` into a URL without encoding that state in DOM attributes or hidden inputs. With Datastar you declare signals once and reference them anywhere in the subtree.

---

## 14. Styling with Tailwind CSS v3

### Setup

```js
// tailwind.config.js
module.exports = {
    content: [
        "./internal/views/**/*.templ",
        "./static/**/*.html",
    ],
    theme: { extend: {} },
    plugins: [],
}
```

The `content` paths tell Tailwind what files to scan for class names. `.templ` files work because Tailwind just looks for class strings — it doesn't need to understand the language.

```css
/* static/css/input.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom utilities */
.sidebar-active {
    @apply bg-indigo-50 text-indigo-700 font-semibold;
}
```

Build:
```bash
./node_modules/.bin/tailwindcss -i static/css/input.css -o static/css/app.css --minify
```

### Component classes

Tailwind in server-rendered HTML is used exactly like in React — classes on elements. The difference is there's no `clsx` or conditional class logic from JS; you use Go's `if` in templ:

```go
templ statusBadge(u models.User) {
    if u.Suspend {
        <span class="bg-red-100 text-red-700 px-2 py-0.5 rounded-full text-xs">Suspended</span>
    } else if u.EmailVerifiedAt != nil {
        <span class="bg-green-100 text-green-700 px-2 py-0.5 rounded-full text-xs">Active</span>
    } else {
        <span class="bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded-full text-xs">Unverified</span>
    }
}
```

---

## 15. Live reload with air

[air](https://github.com/air-verse/air) is a live-reload tool for Go that watches your source files, rebuilds on changes, and restarts the binary. For a stack that has a code-generation step (templ) and an asset build step (Tailwind), the key feature is `pre_cmd` — commands that run before the Go compile on every triggered rebuild.

### Installation

```bash
go install github.com/air-verse/air@latest
```

air installs to `$(go env GOPATH)/bin/air`. Make sure `$GOPATH/bin` is in your `$PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Verify:

```bash
air -v   # v1.xx.x
```

### The full `.air.toml` explained

```toml
# Root of the project (where air looks for .air.toml)
root = "."

# Temporary directory for compiled binaries and build logs.
# Listed in .gitignore. Cleaned on exit when clean_on_exit = true.
tmp_dir = ".air_tmp"

[build]
  # pre_cmd runs BEFORE go build on every rebuild trigger.
  # Commands in this list run sequentially; a non-zero exit aborts the rebuild.
  # This is where the code generation and CSS build live.
  pre_cmd = ["make generate css"]
  #          └─ templ generate ./...
  #             tailwindcss -i static/css/input.css -o static/css/app.css --minify

  # The Go compile command.
  cmd = "go build -o .air_tmp/admin-v2 ./cmd/server"

  # Path to the compiled binary air will execute.
  bin = ".air_tmp/admin-v2"

  # Args forwarded to the binary on startup.
  # e.g. args_bin = ["-config", "dev.yaml"]
  args_bin = []

  # File extensions that trigger a rebuild when changed.
  # "templ" → triggers pre_cmd (templ generate) before Go compile.
  # "css"   → changes to static/css/input.css trigger a rebuild.
  #           app.css is excluded (see exclude_regex) to avoid a loop.
  # "env"   → .env changes restart the server with new env vars loaded.
  include_ext = ["go", "templ", "css", "env"]

  # Directories under root to watch. Limits the inotify/kqueue scope.
  # Don't add "." unless you want air watching node_modules and .git.
  include_dir = ["cmd", "internal", "static"]

  # Directories to never watch — even if inside include_dir.
  exclude_dir  = ["node_modules", ".air_tmp", "bin", "vendor"]

  # Individual files to exclude (glob pattern matched against the basename).
  exclude_file = ["*_templ.go"]

  # Regex matched against the full path of every changed file.
  # Files matching this are silently ignored — critical for templ.
  # Without it: saving accounts.templ → templ writes accounts_templ.go
  #   → air detects accounts_templ.go → runs pre_cmd again → infinite loop.
  exclude_regex = [".*_templ\\.go$"]

  # Time to wait after killing the old binary before starting the new one.
  # 200ms is enough for most Gin servers to release the port.
  kill_delay = "200ms"

  # Path for build error output (separate from stdout so errors are findable).
  log = ".air_tmp/build-errors.log"

  # If true, air stops the watcher loop when the build fails.
  # This prevents a cascade of compile errors from spamming the terminal.
  # Watcher resumes automatically on the next file save.
  stop_on_error = true

[log]
  time  = true    # timestamp each log line
  color = true    # ANSI colours in the terminal

[color]
  # Terminal colours for air's own output (not your app's output).
  main    = "magenta"   # air's header lines
  watcher = "cyan"      # file-watch events
  build   = "yellow"    # build step output
  runner  = "green"     # binary startup/restart messages

[misc]
  # Delete the .air_tmp/ directory when air exits cleanly (Ctrl+C).
  # Prevents stale binaries from being left behind.
  clean_on_exit = true
```

### How the rebuild cycle works

When you save `internal/views/pages/accounts.templ`:

```
1. air inotify/kqueue detects: accounts.templ changed
   (include_ext matches "templ", file is in include_dir, not in exclude_dir)

2. air runs pre_cmd[0]: "make generate css"
   ├── templ generate ./...
   │     Scans all *.templ files
   │     Writes accounts_templ.go  ← IGNORED by watcher (exclude_regex)
   │     Writes other *_templ.go   ← also IGNORED
   └── tailwindcss -i static/css/input.css -o static/css/app.css --minify
         Writes static/css/app.css  ← IS in include_ext "css" BUT...
         air deduplicates: a rebuild is already in progress, so
         the app.css change is folded into the current cycle, not a new one.

3. air runs cmd: "go build -o .air_tmp/admin-v2 ./cmd/server"
   ← Uses the freshly generated *_templ.go files

4. air sends SIGTERM to the running binary (kill_delay = 200ms grace period)

5. air starts .air_tmp/admin-v2
   ← Server is now running the new code

Total wall time: ~1-3s on a modern Mac for a single .templ change
```

When you save a `.go` file (no templ or CSS changes needed):

```
1. air detects: accounts_handler.go changed
2. pre_cmd still runs ("make generate css")
   ← templ generate is a no-op if no .templ files changed
   ← tailwindcss is fast (~200ms) even when nothing changed
3. go build
4. restart
Total: ~0.5-1.5s
```

`pre_cmd` always runs. For a pure Go change this wastes ~200ms on a no-op tailwind pass. If that matters, split the pre_cmd logic to skip CSS when only `.go` files changed — but in practice 200ms is unnoticeable.

### Debugging rebuild problems

**Build errors don't restart the server:**

When `stop_on_error = true` and the build fails, air prints the error and waits. Fix the compile error, save, air rebuilds automatically.

Build errors also go to `.air_tmp/build-errors.log`:

```bash
cat .air_tmp/build-errors.log
```

**Port already in use on restart:**

If the old binary doesn't release the port before the new one starts, increase `kill_delay`:

```toml
kill_delay = "500ms"
```

Or set `SO_REUSEPORT` on your Gin listener. For local dev, 200–500ms is always sufficient.

**air not detecting changes:**

Check `include_dir`. If you add a new directory (e.g. `internal/services/`), air watches it automatically because it's under `internal/` which is already in `include_dir`. But if you create a top-level directory like `pkg/`, you need to add it:

```toml
include_dir = ["cmd", "internal", "static", "pkg"]
```

**templ generate fails silently:**

If `templ` isn't on `$PATH`, `make generate` exits non-zero and air aborts with `stop_on_error = true`. The Makefile resolves this with:

```makefile
TEMPL_CMD=$(shell which templ || echo $(HOME)/go/bin/templ)
```

It falls back to the absolute path in `$GOPATH/bin`.

### Running air

```bash
# From the v2/ directory (air reads .air.toml automatically)
air

# With verbose output
air -d

# Point to a different config
air -c .air.dev.toml
```

### Using multiple air configs

For a project with distinct dev and CI modes:

```toml
# .air.dev.toml — development: pre_cmd includes CSS watch
[build]
  pre_cmd = ["make generate css"]
  include_ext = ["go", "templ", "css", "env"]

# .air.quick.toml — fast iteration: skip CSS rebuild
[build]
  pre_cmd = ["make generate"]
  include_ext = ["go", "templ"]
```

Switch with `air -c .air.quick.toml` when you're only changing Go/templ and don't want the Tailwind pass.

---

## 16. The build pipeline (Makefile)

The Makefile is the single source of truth for all build steps. Every tool — templ, Tailwind, Go — is invoked here, so `make` is the one command that always produces a correct build regardless of what changed.

### Full Makefile

```makefile
# Makefile for admin-v2
# Usage: make [target]

BINARY     = bin/admin-v2
TEMPL_CMD  = $(shell which templ || echo $(HOME)/go/bin/templ)
TAILWIND_CMD = ./node_modules/.bin/tailwindcss

.PHONY: all build generate css run dev clean test tidy help

## all: generate + css + build  (default target)
all: generate css build

## generate: run templ code generation
generate:
	@echo "→ generating templ..."
	$(TEMPL_CMD) generate ./...

## css: build Tailwind CSS
css:
	@echo "→ building Tailwind CSS..."
	$(TAILWIND_CMD) -i static/css/input.css -o static/css/app.css --minify

## build: compile the Go binary
build:
	@echo "→ building binary..."
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server

## run: all + start the server
run: all
	$(BINARY)

## dev: generate + css + go run (no binary, faster iteration)
dev: generate css
	@echo "→ starting dev server on :8080..."
	go run ./cmd/server

## clean: remove all build artifacts
clean:
	rm -rf bin/
	rm -f static/css/app.css
	find . -name '*_templ.go' -delete

## test: run Go tests
test:
	go test ./...

## tidy: tidy Go modules
tidy:
	go mod tidy

## help: print this help message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
```

### Target breakdown

#### `make all` — the default

```makefile
all: generate css build
```

Dependencies execute left-to-right: `generate` → `css` → `build`. This is the correct order because:

1. `templ generate` writes `*_templ.go` files — Go won't compile without them
2. `tailwindcss` writes `static/css/app.css` — the binary serves this at `/static/css/app.css`
3. `go build` compiles everything including the generated `*_templ.go` files

If you skip `generate` and run `go build` directly after adding a new templ component, the build fails with `undefined: pages.NewComponent`. `make all` prevents this class of error.

#### `make generate` — templ code generation

```makefile
TEMPL_CMD = $(shell which templ || echo $(HOME)/go/bin/templ)

generate:
	$(TEMPL_CMD) generate ./...
```

`$(shell which templ || echo $(HOME)/go/bin/templ)` is evaluated at parse time. It resolves `templ` from `$PATH` first, then falls back to the absolute GOPATH location. This makes the Makefile work in CI environments where `go install` writes to `~/go/bin` but that directory isn't in `$PATH`.

`templ generate ./...` scans all packages recursively. For each `*.templ` file it finds, it writes a `*_templ.go` file in the same directory. These files are committed to `.gitignore` — they're generated artifacts, not source.

What gets generated for `accounts.templ`:

```
accounts.templ
    → accounts_templ.go     (the Go implementation of each templ component)
```

The generated code looks like:

```go
// accounts_templ.go — DO NOT EDIT
func Accounts(d AccountsData) templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, templ_7_w io.Writer) (templ_7_err error) {
        // Writes HTML to templ_7_w using Go's io.Writer
        // All output is HTML-escaped unless explicitly marked as safe
        ...
    })
}
```

You can read and debug this code, but any edits are overwritten on the next `make generate`.

#### `make css` — Tailwind build

```makefile
TAILWIND_CMD = ./node_modules/.bin/tailwindcss

css:
	$(TAILWIND_CMD) -i static/css/input.css -o static/css/app.css --minify
```

Using the local `node_modules/.bin/tailwindcss` instead of a global install means every developer and CI run uses the exact version pinned in `package.json`. The `--minify` flag strips whitespace and comments — important for production (saves ~30-60 KB on a typical Tailwind build).

Tailwind scans files listed in `tailwind.config.js content`:

```js
content: [
    "./internal/views/**/*.templ",   // all templ components
    "./static/**/*.html",            // any static HTML
]
```

It generates only the CSS for classes actually used in those files. A fresh `make css` after `make clean` will produce the exact same `app.css` as long as the templ files are unchanged.

**Development tip**: For iterative CSS work, use Tailwind's watch mode directly:

```bash
./node_modules/.bin/tailwindcss -i static/css/input.css -o static/css/app.css --watch
```

This updates `app.css` in-place on every save without a full rebuild. Pair it with `go run ./cmd/server` in a separate terminal for fast CSS iteration.

#### `make build` — Go compile

```makefile
build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server
```

`go build ./cmd/server` compiles the `main` package and all its transitive imports into a single static binary at `bin/admin-v2`. No CGO by default (the MySQL driver uses pure Go).

For a production binary with debug info stripped:

```makefile
build-prod:
	mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/server
```

`-s` strips the symbol table, `-w` strips DWARF debug info. Reduces binary size by ~30%.

#### `make run` vs. `make dev`

```makefile
run: all          # full rebuild + execute the compiled binary
dev: generate css # partial build + go run (no binary written)
    go run ./cmd/server
```

`make run` is for testing the production build path. `make dev` is for quick iteration — `go run` compiles in-memory and skips writing a binary to disk, saving a file-system write. The difference is ~100ms on a fast machine — negligible, but `make dev` is semantically clearer: "I'm developing, not shipping."

For actual hot-reload iteration, use `air` instead of either.

#### `make clean` — full reset

```makefile
clean:
	rm -rf bin/
	rm -f static/css/app.css
	find . -name '*_templ.go' -delete
```

Removes:
- `bin/` — compiled binaries
- `static/css/app.css` — generated Tailwind output
- All `*_templ.go` files anywhere in the tree — generated templ code

After `make clean`, running `make all` rebuilds everything from scratch. Use this when:
- Upgrading the `templ` version (generated code format may change)
- Upgrading Tailwind (utility class names may change)
- Debugging a "stale generated file" issue
- Onboarding a new developer who just cloned the repo

#### `make test` and `make tidy`

```makefile
test:
	go test ./...   # runs all tests in the module

tidy:
	go mod tidy     # removes unused deps, adds missing ones
```

Run `make tidy` after adding or removing imports. It updates both `go.mod` and `go.sum`. Commit both files.

### TEMPL_CMD resolution in CI

In GitHub Actions or similar CI without `$GOPATH/bin` on `$PATH`:

```yaml
# .github/workflows/build.yml
- name: Install templ
  run: go install github.com/a-h/templ/cmd/templ@latest

- name: Build
  run: make all   # TEMPL_CMD falls back to $HOME/go/bin/templ
```

The `$(shell which templ || echo $(HOME)/go/bin/templ)` expansion handles this automatically.

### Adding a watch target (without air)

If you don't want to install air but want file watching, add a `watch` target using `entr`:

```makefile
watch:
	find . -name '*.go' -o -name '*.templ' -o -name 'input.css' \
	  | entr -r make dev
```

`brew install entr` on macOS. `entr -r` re-runs the whole command on any change. This is cruder than air (no debounce, no partial rebuild), but works without an extra Go install.

### Parallel builds (advanced)

For large projects where `templ generate` and other pre-steps can run in parallel:

```makefile
prepare:
	$(TEMPL_CMD) generate ./... &  \
	$(TAILWIND_CMD) -i static/css/input.css -o static/css/app.css --minify & \
	wait

all: prepare build
```

This runs templ and Tailwind concurrently, then waits for both before compiling. For admin-v2 the sequential approach is fast enough (~2s total), but for larger templ codebases this can save 1-2s per rebuild.

---

## 17. End-to-end walkthrough: the accounts list

This section traces a single request through every layer.

### 1. Request arrives: `GET /accounts?q=john&page=2`

Gin matches the route to `h.GetAccounts` (registered in `RegisterRoutes` under the `protected` group).

### 2. `middleware.SessionAuth()` runs first

It checks `c.Get(AdminIDKey)` — set by `LoadSessionAdmin()` earlier in the chain. If missing, redirects to `/login`.

### 3. Handler builds the filter

```go
func (h *Handler) GetAccounts(c *gin.Context) {
    f := buildUserFilter(c, nil)
    roleID := uint(1)
    f.RoleID = &roleID

    users, total, err := h.UserStore.ListUsers(f)
    // ...
}
```

`buildUserFilter` parses `?q=john&page=2` from the query string. `f.RoleID = &roleID` pins the query to accounts (role 1), excluding reps and affiliates.

### 4. Store executes two SQL queries

```go
func (s *UserStore) ListUsers(f UserFilter) ([]models.User, int64, error) {
    where, args := buildUserWhere(f)
    // WHERE u.deleted_at IS NULL AND u.user_role_id = 1 AND (u.name LIKE ? OR u.email LIKE ? ...)

    var total int64
    s.db.Get(&total, `SELECT COUNT(*) FROM users u LEFT JOIN user_roles ur ... `+where, args...)

    offset := (f.Page - 1) * f.Limit  // (2-1)*50 = 50
    listArgs := append(args, f.Limit, offset)
    var users []models.User
    s.db.Select(&users, `SELECT ... LIMIT ? OFFSET ?`, listArgs...)

    return users, total, nil
}
```

`sqlx.Select` scans all rows into `[]models.User` using the `db:` struct tags.

### 5. Handler renders the template

```go
c.Header("Content-Type", "text/html; charset=utf-8")
pages.Accounts(pages.AccountsData{
    Users: users, TotalItems: total,
    Page: f.Page, Limit: f.Limit, Search: f.Search,
}).Render(c.Request.Context(), c.Writer)
```

`Render` writes HTML directly to `c.Writer`. No buffering.

### 6. templ renders the component tree

```
Accounts(AccountsData)
  → layouts.Dashboard("Accounts")
      → layouts.Base("Accounts")
          → <html><head>...</head><body>
              → <div class="flex h-full">
                  → Sidebar()
                  → Topbar("Accounts")
                  → <main>
                      → toolbar with Datastar search form
                      → <div id="accounts-table">
                          → AccountsTable(d)
                              → <table>... for _, u := range d.Users { <tr>... } </table>
                              → Pagination(...)
```

### 7. Browser loads the page

The page includes the Datastar script. The search form has:

```html
<form
    data-store='{"q": ""}'
    data-on-input__debounce.300ms="@get('/accounts/search?q=' + $q)">
    <input type="text" name="q" value="john" data-bind="q" .../>
</form>
```

### 8. User types in the search box

After 300ms idle, Datastar fires `GET /accounts/search?q=jo`. The server returns just `AccountsTable` HTML. Datastar replaces the content of `#accounts-table`.

The URL bar doesn't change — this is an in-place DOM update, not a navigation.

---

## 18. Patterns and tradeoffs

### What works well

**templ over html/template**. Type-safe components with real Go control flow catch missing data at compile time, not at runtime. The generated code is readable Go — you can set breakpoints in it.

**sqlx over GORM**. You own the queries. N+1 issues are visible, not hidden behind `Preload()`. The `UserFilter` pattern makes complex dynamic queries explicit and testable.

**Dual auth layers**. Session cookies for the browser, JWT for the API. Both layers use the same `AdminIDKey` in the Gin context, so middleware that checks `c.Get(AdminIDKey)` works transparently for both. Adding new routes doesn't require thinking about which auth layer to use — they coexist.

**Datastar for selective reactivity**. Use it only where you'd otherwise need a full-page reload that hurts UX: live search, inline form submission feedback, toggles. Static pages stay static. You don't have to decide upfront whether the whole page is "reactive."

**air with `pre_cmd`**. Running the full build pipeline (templ → CSS → Go compile) on every save is slower than hot-module replacement but is completely reliable — what you see in the browser is always the compiled output of the current source.

### Tradeoffs

**templ requires a code generation step.** `make generate` must run before `go build`. In CI, this means two steps. The `Makefile` handles it with `make all`. air handles it via `pre_cmd`. But it's one more thing to document.

**Datastar is not HTMX.** It's newer, smaller, and more opinionated about signals. The SSE-based merge is more powerful than HTMX's `hx-swap`, but the ecosystem and Stack Overflow answers are thinner. Budget time to read the Datastar docs directly.

**No read/write connection split in the current `main.go`.** The `WritableConnect()` function exists in `store/db.go` but `main.go` calls only `Connect()` for all stores. To add a read replica, pass separate pools to stores that only read vs. stores that write.

**PDF merging is naive.** `APIGetLabel` concatenates PDF bytes — this works for single-page labels but breaks on multi-page PDFs. Replace with `pdfcpu` (pure Go) or `unipdf` (commercial, comprehensive) before shipping label download to production.

### Extending the architecture

**Adding a new page:**

1. Add a templ component in `internal/views/pages/newpage.templ`
2. Add a handler method on `*Handler` in a new or existing handler file
3. Add the route in `routes.go`
4. Run `make generate` (or let air do it)

**Adding a new store method:**

1. Add the method to the relevant `*Store` struct
2. The method is immediately available in any handler via `h.UserStore.NewMethod()`

**Adding a new API endpoint:**

1. Write the handler as a method on `*Handler`
2. Register it under `api := r.Group("/api", middleware.RequireAuth())` in `routes.go`
3. Return JSON with `c.JSON(status, payload)`

No codegen, no DTO transformers, no reflection magic.

---

## 19. Performance: skeleton-first pages + parallel queries

The original pages did all their database work **before** sending a single byte of HTML. On a slow query, the browser stared at a blank white screen the whole time. This section covers the two techniques that fixed it — and the reasoning behind each.

### The problem: blocking render

Here's what the dashboard handler used to do:

```go
// SLOW — the browser waits for all 4 queries before ANY HTML arrives
func (h *Handler) GetHome(c *gin.Context) {
    start, end := dashboardRange()
    newUsers, _    := h.OrderStore.QueryActiveUsersPerformance(start, end) // wait…
    newLabels, _   := h.OrderStore.QueryNewLabels(start, end)              // wait…
    printPerf, _   := h.OrderStore.QueryPrintPerformance(start, end)       // wait…
    activeUsers, _ := h.OrderStore.QueryActiveUsersPerformance(start, end) // wait…
    // Only NOW does any HTML render
    pages.Home(...).Render(c.Request.Context(), c.Writer)
}
```

If each query takes 400ms, the user waits 1.6 seconds looking at nothing. Two separate problems are hiding here:

1. **Blocking render** — no HTML is sent until every query finishes.
2. **Sequential queries** — the four queries run one after another, even though they don't depend on each other.

We fix each one with a different technique.

### Technique 1: skeleton-first with lazy fragment loading

The idea: send the page **shell** instantly (no DB work), with placeholder "skeleton" boxes where the data will go. Each placeholder then fetches its own data in the background and swaps itself out when ready.

This is what Gmail, LinkedIn, and most modern apps do — you see the layout immediately, then content fills in. It makes the app *feel* fast even when the queries aren't.

**Step 1 — the reusable skeleton components** (in `internal/views/layouts/base.templ`):

```go
// LazySection renders a placeholder that fetches its real content from `src`
// as soon as it appears in the DOM. Datastar's data-on-load fires the request.
templ LazySection(id, src string) {
    <div id={ id } data-on-load={ "@get('" + src + "')" }>
        @SkeletonCard()
    </div>
}

// A shimmering grey placeholder (Tailwind's animate-pulse does the shimmer).
templ SkeletonCard() {
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5 animate-pulse">
        <div class="h-3 w-1/3 bg-gray-200 rounded mb-4"></div>
        <div class="space-y-2">
            <div class="h-3 w-full bg-gray-100 rounded"></div>
            <div class="h-3 w-5/6 bg-gray-100 rounded"></div>
        </div>
    </div>
}
```

`data-on-load` is a Datastar trigger that fires **once** when the element enters the DOM. So the moment the shell paints, every `LazySection` simultaneously fires a `@get(...)` to its fragment URL.

**Step 2 — the handler renders only the shell** (zero DB work):

```go
// GetHome renders the shell only — paints in ~5ms.
func (h *Handler) GetHome(c *gin.Context) {
    c.Header("Content-Type", "text/html; charset=utf-8")
    pages.Home().Render(c.Request.Context(), c.Writer)  // no query here!
}
```

**Step 3 — the shell page is a grid of placeholders:**

```go
templ Home() {
    @layouts.Dashboard("Dashboard") {
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            @layouts.LazySection("stat-new-users", "/dashboard/stat/new-users")
            @layouts.LazySection("stat-new-labels", "/dashboard/stat/new-labels")
            @layouts.LazySection("stat-printed", "/dashboard/stat/printed")
            @layouts.LazySection("stat-active", "/dashboard/stat/active-shippers")
        </div>
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            @layouts.LazySection("tbl-new-users", "/dashboard/table/new-users")
            @layouts.LazySection("tbl-new-labels", "/dashboard/table/new-labels")
            // ...
        </div>
    }
}
```

**Step 4 — each fragment endpoint runs its own query and returns just that section:**

```go
func (h *Handler) DashStatNewUsers(c *gin.Context) {
    start, end := dashboardRange()
    rows, err := h.OrderStore.QueryActiveUsersPerformance(start, end)
    h.renderStat(c, "stat-new-users", "New Users (latest week)", latestTotal(rows), "indigo", err)
}

// renderStat sends back a single card whose root id matches the placeholder.
func (h *Handler) renderStat(c *gin.Context, id, label, value, color string, err error) {
    c.Header("Content-Type", "text/html; charset=utf-8")
    if err != nil {
        value = "—"  // one failed query never blanks the whole page
    }
    pages.StatCardFragment(id, label, value, color).Render(c.Request.Context(), c.Writer)
}
```

The fragment's root element carries the **same `id`** as its placeholder (`stat-new-users`), so Datastar knows which DOM node to replace:

```go
templ StatCardFragment(id, label, value, color string) {
    <div id={ id } class="bg-white rounded-xl border ...">
        <p class="text-xs ...">{ label }</p>
        <p class={ "text-2xl font-bold", colorClass(color) }>{ value }</p>
    </div>
}
```

**The result:** the page paints instantly with 8 shimmering boxes. The browser fires 8 parallel requests. Each box pops in the moment its query returns — the fast ones (a cached stat) appear first, the slow ones (a big aggregation) a moment later. The user never sees a blank screen.

**Why the `id` must match:** Datastar's default merge finds the element in the response whose `id` matches an element already in the DOM, and swaps it. If the fragment's root `id` didn't match the placeholder's `id`, nothing would swap. This is the single most common mistake with this pattern.

### Technique 2: parallel queries with goroutines

The skeleton pattern helps when a page has several *independent sections*. But sometimes a **single** endpoint genuinely needs several queries before it can respond. Running them sequentially wastes time when they don't depend on each other.

Go's goroutines make parallel queries easy. The label report endpoint needs two independent queries — the main service report and the return-label report:

```go
// BEFORE — sequential: total time = queryA + queryB
mainRows, _ := h.OrderStore.QueryLabelReportRows(start, end, mainIDs, userID)
returnRows, _ := h.OrderStore.QueryReturnLabelReportRows(start, end, userID)
```

Rewritten to run both at once with `errgroup`:

```go
import "golang.org/x/sync/errgroup"

func (h *Handler) APIReportLabels(c *gin.Context) {
    // ... parse request ...

    var mainRows, returnRows []store.LabelReportRow

    // errgroup runs each g.Go(...) in its own goroutine and collects errors.
    g, _ := errgroup.WithContext(c.Request.Context())

    g.Go(func() error {
        rows, err := h.OrderStore.QueryLabelReportRows(req.StartDate, req.EndDate, mainIDs, req.UserID)
        if err != nil {
            return err
        }
        mainRows = rows   // each goroutine writes its OWN variable — no data race
        return nil
    })

    g.Go(func() error {
        rows, err := h.OrderStore.QueryReturnLabelReportRows(req.StartDate, req.EndDate, req.UserID)
        if err != nil {
            return err
        }
        returnRows = rows
        return nil
    })

    // Wait blocks until BOTH goroutines finish; returns the first error, if any.
    if err := g.Wait(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Safe to merge now — both goroutines have finished.
    results := map[string]models.LabelReport{}
    for _, r := range append(mainRows, returnRows...) {
        // ... build result ...
    }
    c.JSON(http.StatusOK, results)
}
```

Now total time is `max(queryA, queryB)` instead of `queryA + queryB`.

**What is `errgroup`?** It's a small helper from `golang.org/x/sync`. You call `g.Go(func)` for each concurrent task, then `g.Wait()` blocks until they all finish. If any returns an error, `Wait()` returns the first one. It's the idiomatic Go way to run a fixed set of parallel tasks and collect their results.

### The critical rule: no shared writes

The single most important thing to understand about goroutines: **two goroutines must never write the same variable at the same time.** That's a *data race* — it corrupts memory and causes crashes that only appear under load.

Notice how the code above avoids it: each goroutine writes its **own** variable (`mainRows` vs `returnRows`). They never touch the same memory. The merge into the shared `results` map happens *after* `g.Wait()`, when both goroutines have finished and only the main goroutine is running.

The PDF-fetch code uses the same trick with a pre-sized slice, so concurrent fetches write distinct indices:

```go
fetched := make([][]byte, len(req.URLs))
var wg sync.WaitGroup
for i, url := range req.URLs {
    wg.Add(1)
    go func(idx int, u string) {
        defer wg.Done()
        data, err := fetchURL(u)
        if err == nil {
            fetched[idx] = data  // each goroutine writes a DISTINCT index — safe
        }
    }(i, url)
}
wg.Wait()  // wait for all fetches
```

`sync.WaitGroup` is the lower-level primitive `errgroup` is built on: `Add(n)` to count tasks, `Done()` as each finishes, `Wait()` to block until the counter hits zero. Use `errgroup` when tasks can error; use `WaitGroup` when they can't (like a best-effort fetch that just skips failures).

**Why the loop variable is passed as an argument:** `go func(idx int, u string){...}(i, url)` — the `i` and `url` are passed *into* the goroutine as arguments rather than captured. In older Go this was mandatory to avoid every goroutine seeing the final loop value. As of Go 1.22 the loop variable is per-iteration so capture is safe too, but passing explicitly is still the clearest style.

### Does the connection pool handle this?

Yes. Recall from Section 6 that `db.SetMaxOpenConns(25)` lets up to 25 queries run against MySQL simultaneously. Firing 2–4 concurrent queries per request is well within that budget. `sqlx` is safe for concurrent use — each goroutine gets its own connection from the pool automatically.

### When to reach for each technique

| Situation | Technique |
|---|---|
| Page has several independent sections | Skeleton + lazy `LazySection` per section |
| One endpoint needs several independent queries | `errgroup` parallel goroutines |
| Fetching many URLs / files | `sync.WaitGroup` + pre-sized slice |
| A single fast query | Neither — just render synchronously |

Don't over-apply this. The login page and settings page still render synchronously — they're fast and adding lazy-loading would only add complexity. Use these patterns where there's a real latency problem to solve.

---

## 20. End-to-end testing with Playwright

Unit tests check one function. **End-to-end (e2e) tests** check the whole app the way a real user does: they open a browser, click buttons, fill forms, and assert on what appears on screen. For a server-rendered app like this, e2e tests are especially valuable — they catch broken templates, bad routes, and auth bugs that a unit test would miss.

[Playwright](https://playwright.dev) is a browser-automation library from Microsoft. It drives a real Chromium/Firefox/WebKit browser from code.

### Why e2e tests matter for this app

The lazy-loading change from Section 19 is a perfect example. A unit test on `DashStatNewUsers` proves the handler returns the right HTML. But only an e2e test proves the **whole chain** works: the shell renders, Datastar fires the `data-on-load` request, the fragment comes back, and the skeleton actually gets replaced by real data in the browser. That integration is exactly where bugs hide.

### Setup

Playwright is a Node tool (there's also a Go port, but the Node version has the richest tooling). Install it in a `tests/` directory:

```bash
mkdir -p tests && cd tests
npm init -y
npm install -D @playwright/test
npx playwright install chromium   # downloads the browser binary
```

Create `tests/playwright.config.ts`:

```ts
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  use: {
    baseURL: 'http://127.0.0.1:8080',  // the Go server
    // Capture a trace on first retry — invaluable for debugging failures
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  // Start the Go server automatically before the tests run
  webServer: {
    command: 'cd .. && make run',
    url: 'http://127.0.0.1:8080/ping',
    reuseExistingServer: true,
    timeout: 30_000,
  },
});
```

The `webServer` block is the key convenience: Playwright boots your Go server, waits for `/ping` to answer, runs the tests, then shuts it down.

### A first test: the login page renders

Create `tests/e2e/login.spec.ts`:

```ts
import { test, expect } from '@playwright/test';

test('login page shows the sign-in form', async ({ page }) => {
  await page.goto('/login');

  // The heading is visible
  await expect(page.getByRole('heading', { name: 'GrandShipper Admin' })).toBeVisible();

  // Email and password fields exist
  await expect(page.getByPlaceholder('admin@example.com')).toBeVisible();
  await expect(page.locator('input[name="password"]')).toBeVisible();

  // The submit button says "Sign in"
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});

test('protected routes redirect to login when logged out', async ({ page }) => {
  await page.goto('/accounts');
  // SessionAuth middleware bounces us to /login
  await expect(page).toHaveURL(/\/login$/);
});
```

Run it:

```bash
npx playwright test
```

The second test proves the auth middleware from Section 11 works — visiting `/accounts` logged-out lands you on `/login`.

### Testing the login flow

```ts
test('valid credentials log the user in', async ({ page }) => {
  await page.goto('/login');

  await page.getByPlaceholder('admin@example.com').fill('admin@grandshipper.com');
  await page.locator('input[name="password"]').fill('test-password');
  await page.getByRole('button', { name: 'Sign in' }).click();

  // After login we land on the dashboard
  await expect(page).toHaveURL('http://127.0.0.1:8080/');
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
});

test('bad credentials show an error', async ({ page }) => {
  await page.goto('/login');
  await page.getByPlaceholder('admin@example.com').fill('nobody@example.com');
  await page.locator('input[name="password"]').fill('wrong');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page.getByText("Email or password don't match")).toBeVisible();
});
```

> **Test data:** never test against production. Point `DATABASE_URL` at a local or throwaway database seeded with known test accounts. The `webServer.command` in the config can set it: `DATABASE_URL=... make run`.

### Testing the lazy-loading dashboard

This is the test that proves Section 19 actually works end to end. We need a **logged-in** session first, so we use a `beforeEach` hook:

```ts
import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  // Log in before each test in this file
  await page.goto('/login');
  await page.getByPlaceholder('admin@example.com').fill('admin@grandshipper.com');
  await page.locator('input[name="password"]').fill('test-password');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL('http://127.0.0.1:8080/');
});

test('dashboard skeletons are replaced by real data', async ({ page }) => {
  // The shell renders 8 skeleton placeholders instantly.
  // Datastar then fetches each fragment and swaps it in.

  // Wait for a stat card to load its real content.
  // The fragment's root keeps id="stat-new-users"; once loaded it holds a value.
  const newUsersCard = page.locator('#stat-new-users');
  await expect(newUsersCard).toBeVisible();

  // The skeleton has the animate-pulse class; the real card does not.
  // Wait until the shimmer is gone (data has arrived).
  await expect(newUsersCard).not.toHaveClass(/animate-pulse/);

  // The card shows its label
  await expect(newUsersCard).toContainText('New Users');

  // All four stat cards eventually load
  for (const id of ['#stat-new-users', '#stat-new-labels', '#stat-printed', '#stat-active']) {
    await expect(page.locator(id)).not.toHaveClass(/animate-pulse/);
  }
});
```

The assertion `not.toHaveClass(/animate-pulse/)` is the crux: the skeleton *has* that class, the loaded fragment does *not*. Playwright's `expect` auto-retries for a few seconds, so it naturally waits for the async fragment to arrive and swap in — no manual `sleep` needed.

### Testing debounced search

The accounts search fires a Datastar request 300ms after you stop typing, swapping the table fragment:

```ts
test('typing in accounts search filters the table', async ({ page }) => {
  await page.goto('/accounts');

  // Wait for the initial table to load
  await expect(page.locator('#accounts-table table')).toBeVisible();

  // Type a search term
  await page.getByPlaceholder('Search accounts…').fill('smith');

  // Datastar debounces 300ms then GETs /accounts/search?q=smith
  // and swaps #accounts-table. Playwright waits for the result.
  await expect(page.locator('#accounts-table')).toContainText('smith', {
    ignoreCase: true,
    timeout: 5000,
  });
});
```

### Useful Playwright commands

```bash
npx playwright test                    # run all tests headless
npx playwright test --headed           # watch the browser drive itself
npx playwright test login.spec.ts      # run one file
npx playwright test --ui               # interactive UI mode — great for debugging
npx playwright show-report             # open the HTML report after a run
npx playwright codegen localhost:8080  # RECORD a test by clicking through the app
```

`codegen` is the best way to start: it opens a browser, records your clicks and types, and writes the test code for you. You then clean it up and add assertions.

### How Playwright waits (and why you rarely need sleep)

The most common mistake coming from other tools is sprinkling `sleep(1000)` everywhere. Playwright's `expect(...).toBeVisible()` and locator actions **auto-wait** — they retry for up to 5 seconds (configurable) until the condition is met. For an app with async fragment loading like ours, this is exactly right: the test waits for the fragment to arrive without you hard-coding a delay. Only reach for an explicit wait (`page.waitForResponse('/dashboard/stat/new-users')`) when you need to assert on the network request itself.

### CI integration

Playwright runs headless in CI with no extra setup. A GitHub Actions job:

```yaml
- uses: actions/setup-node@v4
  with: { node-version: 20 }
- uses: actions/setup-go@v5
  with: { go-version: '1.26' }
- run: go install github.com/a-h/templ/cmd/templ@latest
- run: cd tests && npm ci && npx playwright install --with-deps chromium
- run: cd tests && npx playwright test
```

The `webServer` config boots the Go app, so CI needs only a test database. On failure, Playwright saves a trace you can open locally with `npx playwright show-trace trace.zip` — it's a full timeline of the run with DOM snapshots at every step.

---

*This tutorial is derived from the actual `admin-v2` source code. All code examples are taken directly from the running application.*
