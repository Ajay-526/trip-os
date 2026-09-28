# Trip OS

An India-first group travel operating system: create a trip, invite your
people, everyone states preferences, an AI planner proposes a structured,
deterministically-validated itinerary, the group votes, bookings are tracked
by reference, and everything collapses into a single live Trip Book.

This delivery implements **Phase 0 of the MVP** (see
`docs/implementation-plan.md`): repository setup, database schema, auth,
trip + membership domain, trip creation UI, invite/join workflow, and the
group preference workflow — as real, compiling, tested code. AI planning,
itinerary validation, voting, bookings, and the Trip Book are specified in
full (architecture, schema, API contract) but scaffolded as the next phase,
per the instruction to build the MVP incrementally rather than everything
at once.

## Read first

- `architecture.md` — system design and the "LLM is not the source of
  truth" pipeline
- `docs/database-schema.sql` — full MVP schema
- `docs/api-contract.md` — full MVP REST contract
- `docs/implementation-plan.md` — all 20 increments, sequenced
- `docs/environment-variables.md` — every env var, backend and frontend
- `docs/test-strategy.md` — what's tested at each layer and why

## Running it locally

### 1. Infrastructure

```bash
docker compose up -d          # Postgres on :5432, Redis on :6379
```

### 2. Backend (Go)

```bash
cd backend
cp .env.example .env          # edit if you changed docker-compose credentials
go run ./cmd/api              # http://localhost:8080
```

Migrations: apply `migrations/0001_init.up.sql` with `psql` or the
`golang-migrate` CLI (`make migrate-up` — see `backend/Makefile`).

Run the test suite:

```bash
cd backend
go test ./... -cover
```

### 3. Frontend (Next.js)

```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev                   # http://localhost:3000
```

## Deploy to a free test environment

I can't host this myself (no inbound networking or hosting-provider access
from where I run), but this is set up so you can have it live in about ten
minutes on free tiers: **Render** for the Postgres + Go API, **Vercel** for
the Next.js frontend. Both configs are already in the repo.

### 1. Push this to a GitHub repo

Render and Vercel both deploy from a git repo you own.

The project already contains a git repo with an initial commit. Create an
empty repo on GitHub, then:

```bash
cd trip-os
git remote add origin git@github.com:<you>/trip-os.git
git push -u origin main
# or, with the GitHub CLI: gh repo create trip-os --private --source=. --push
```

### 2. Backend + database on Render

1. Go to [dashboard.render.com/blueprints](https://dashboard.render.com/blueprints) → **New Blueprint Instance** → pick your `trip-os` repo. Render reads `render.yaml` at the repo root and provisions a free Postgres instance plus the API as a Docker web service, with `JWT_ACCESS_SECRET` and `CSRF_SECRET` auto-generated.
2. Wait for the first deploy to go green, then open a **Shell** on the `trip-os-api` service (or connect with `psql` using the connection string from the `trip-os-db` dashboard page) and apply the schema once:
   ```bash
   psql "$DATABASE_URL" -f backend/migrations/0001_init.up.sql
   ```
3. Note the API's public URL, e.g. `https://trip-os-api.onrender.com`.

*(Render's free Postgres is deleted after 30 days of inactivity/on a timer — fine for testing, not for anything you want to keep.)*

### 3. Frontend on Vercel

1. [vercel.com/new](https://vercel.com/new) → import the same repo → set **Root Directory** to `frontend`. `vercel.json` handles the rest.
2. Add an environment variable: `NEXT_PUBLIC_API_BASE_URL` = `https://trip-os-api.onrender.com/api/v1` (your Render URL from step 2, plus `/api/v1`).
3. Deploy. Note the resulting URL, e.g. `https://trip-os.vercel.app`.

### 4. Connect the two

Cookies and CORS are locked to an exact origin on purpose (see
`internal/middleware/cors.go`), so the backend needs to know the frontend's
real URL:

1. Back in Render, open `trip-os-api` → **Environment** → set `ALLOWED_ORIGIN` to your exact Vercel URL from step 3 (no trailing slash) → save (this redeploys).
2. `CROSS_SITE_COOKIES=true` and `COOKIE_SECURE=true` are already set by `render.yaml` — that's what makes the session cookie valid across the two domains.

Open the Vercel URL, sign up, create a trip, generate an invite link, and
open it in a second browser (or incognito window) to test the join flow as
a second person.

### Keeping it running

Render's free web service spins down after 15 minutes idle and takes
10-20s to wake back up on the next request — expect that cold-start delay
if you come back to test after a break. Everything above is disposable and
free; tear it down from each dashboard when you're done.

## What's real vs. what's scaffolded

**Implemented and tested:**
- Signup/login/logout/refresh with httpOnly cookies, Argon2id password
  hashing, rotating refresh tokens, CSRF double-submit protection
- Trip creation, retrieval, update, soft delete, with role-based
  authorization (non-members get 404, wrong role gets 403)
- Email invites and shareable multi-use invite links, join flow
- Per-member preferences (budget, pace, interests, dietary, time windows),
  visible in aggregate to organizers only
- A mobile-first Next.js UI for all of the above, with explicit
  loading/error/empty states throughout

**Specified, not yet built** (see `docs/implementation-plan.md` phases 1–3):
place/planning/transport provider adapters, the itinerary structured-output
schema and deterministic validator, itinerary UI, group voting, booking
references, the Trip Book aggregate, notifications, analytics, admin, and
production observability.

## A note on the Go module setup

This environment's network egress doesn't reach `proxy.golang.org`, so
`go.mod` routes a few transitive dependencies (`golang.org/x/*`,
`gopkg.in/yaml.v3`, `gopkg.in/check.v1`) to their GitHub mirrors via
`replace` directives instead. Functionally identical to the canonical
modules — just fetched a different way. On a machine with normal network
access these replace directives are harmless but unnecessary; feel free to
remove them and re-run `go mod tidy` there.
