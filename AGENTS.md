# cifra-finops — AI Conventions

Primary reader is often an LLM agent. Prefer small greps, small modules, and headless tests.

## Rule #1: TDD is mandatory

Before writing production code:

1. Write the test (`*_test.go`, or `web/src/pages/*.test.js` for Svelte)
2. Run focused: `go test ./... -v -run TestName` / `npm run test:fe`
3. Confirm it **fails** for the right reason
4. Write the **minimal** code to pass
5. Run `cais test` (and frontend tests when JS changed)

## Clean code for agents

| Priority | Rule                                                                                                   |
| -------- | ------------------------------------------------------------------------------------------------------ |
| 1        | **Small units** — functions ~4–20 lines; files target 200–300 lines, hard cap ~500                     |
| 2        | **SRP** — one reason to change per file/package                                                        |
| 3        | **Greppable names** — unique domain nouns; avoid `data`, `handler`, `Manager`, `util` as primary names |
| 4        | **Comments = WHY** — security, SQLite, CSRF/cookie, Inertia JSON vs form. No narrating WHAT            |
| 5        | **Inject deps** — handlers take `Store`, Inertia, `cais.Config` via constructor                        |
| 6        | **Early returns** — max ~2 nesting levels                                                              |
| 7        | **Errors with values** — `fmt.Errorf("...: %w", err)`                                                  |
| 8        | **Headless tests** — SQLite `:memory:`; no manual seed for unit tests                                  |

## Layout

| Path                     | Responsibility               |
| ------------------------ | ---------------------------- |
| `cmd/server/`            | Entry point                  |
| `internal/app/`          | Bootstrap, `registerRoutes`  |
| `internal/handlers/`     | HTTP handlers (`view.Write`) |
| `internal/store/`        | SQLite + migrations          |
| `internal/models/`       | Domain structs               |
| `web/templates/app.html` | Inertia root shell           |
| `web/src/pages/`         | Svelte pages                 |
| `web/src/components/`    | Shared Svelte components     |
| `web/static/`            | CSS, Vite build, PWA         |

Patch markers (do not remove): `registerRoutes`, `Close() error`, `<!-- cais:nav -->`.

## Inertia + Svelte

Handlers render **Amarra HTML** (`view.Write`). Inertia + Svelte is gone.

```go
_ = h.inertia.Render(w, r, "Contact", inertia.Props{"site": meta.ForRequest(h.site, r)})

// Validation — re-render same component
ve := make(inertia.ValidationErrors)
ve["email"] = "Invalid email"
ctx := inertia.SetValidationErrors(r.Context(), ve)
_ = h.inertia.Render(w, r.WithContext(ctx), "Contact", inertia.Props{})

// Flash on redirect — cais cookie API only (not inertia.SetFlash)
flash.Set(w, "notice", "Saved!", cfg.CookieSecure())
h.inertia.Redirect(w, r, "/dashboard", http.StatusSeeOther)
// Next request: middleware.Flash → flash.MessageFromRequest → props["flash"]
```

Svelte pages (`@inertiajs/svelte` + Svelte 5):

- `useForm` returns a **reactive object**, not a store — no `$form`
- Do **not** reactive-write props into the form (`$: form.x = prop`) — can blank the page
- Prefer local state for derived UI; assign into form on submit
- Password fields: `PasswordInput.svelte`
- Mutations: `form.post` / `router.post` (`use:inertia` is for GET only)

Parse bodies with `httpx.ParseFormOrJSON` (Inertia posts JSON).

## Auth, CSRF, flash

- Session middleware: `LoadSession` + `Flash` + `CSRF(cfg)`
- Protect routes: `middleware.RequireAuth("/login")` / `RequireAuthFunc`
- CSRF: double-submit cookie `cais_csrf` + form field or `X-CSRF-Token`
- Flash: **only** `flash.Set` + read via `flash.MessageFromRequest` into Inertia props
- Dev demo user (when seeded): `demo@example.com` / `password`

## New page / resource

```bash
cais g handler settings     # handler + test + web/src/pages/Settings.svelte + route
cais g page about           # Svelte page only
cais g resource bookmark --fields title:string,url:url,notes:text?
cais g model tag --fields name:string
cais g migration add_notes
cais g auth                 # if app was --blank/--minimal
cais db migrate
```

Or by hand:

1. Go test in `internal/handlers/`
2. Optional Vitest in `web/src/pages/*.test.js`
3. Svelte page + handler + route in `internal/app/routes.go`

## Commands

```bash
cais install          # npm + go mod tidy (+ Tailwind build)
cais dev              # air + tailwind + vite watch
cais test             # go test ./...
npm run test:fe       # Vitest (Svelte)
make ci               # test + lint + format-check
cais doctor [--mobile]
cais routes
cais db migrate | status | rollback | seed
```

## Do not

- Parse templates per request
- Use inline CSS (Tailwind classes)
- Mock the database (use SQLite `:memory:`)
- Grow files past ~500 lines without splitting
- Ship features without a headless test
- Use `$form` store syntax or Svelte 4 `new App()`
- Call `inertia.SetFlash` (no-op without FlashDataProvider — use `flash.Set`)
- Reactive-assign Inertia props into `useForm` fields
