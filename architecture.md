# Trip OS — Architecture

## 1. Product framing

Trip OS is a group-travel operating system: a trip has members, members express
preferences, an AI planner proposes a structured itinerary grounded in real
place/destination data, the group votes on options, bookings are tracked by
reference (not executed by us in MVP), and the result collapses into a single
live "Trip Book" — the source of truth for the trip.

## 2. Non-negotiable architectural rule

**The LLM is never the source of truth.**

Pipeline for any AI-assisted output:

```
1. Retrieve   -> deterministic calls to place/maps/transport providers,
                 constrained to what actually exists (place IDs, opening
                 hours, geo-coordinates, categories).
2. Ground     -> assemble that data into a structured context/tool payload.
3. Generate   -> LLM produces a response constrained to a JSON Schema
                 (structured output / tool-call, not free text).
4. Validate   -> deterministic Go code re-checks the LLM's output against
                 the same source data: place IDs must exist, times must be
                 inside opening hours, no overlapping blocks, travel time
                 between consecutive stops must be feasible, per-day/total
                 budget must not be exceeded.
5. Persist/Show -> only validated itineraries are stored or rendered. Invalid
                 output is either auto-repaired deterministically (e.g. drop
                 an infeasible stop) or rejected with a re-generation loop
                 (bounded retries), never shown raw.
```

The LLM call is one interchangeable adapter behind a `PlanningProvider`
interface. Swapping the model or provider never touches domain or validation
code.

## 3. High-level system diagram

```
                    ┌─────────────────────────┐
                    │   Next.js (TypeScript)  │
                    │  Mobile-first web app    │
                    └────────────┬─────────────┘
                                 │ HTTPS / JSON (REST)
                                 ▼
                    ┌─────────────────────────┐
                    │        Go API            │
                    │  (modular monolith)      │
                    │                          │
                    │  cmd/api                 │
                    │  internal/domain/*       │  <- entities + business rules
                    │  internal/service/*      │  <- use cases / orchestration
                    │  internal/handler/*      │  <- HTTP transport (chi)
                    │  internal/repository/*   │  <- Postgres persistence
                    │  internal/provider/*     │  <- external adapters
                    │  internal/validator/*    │  <- deterministic itinerary checks
                    └───────┬─────────┬────────┘
                            │         │
                 ┌──────────┘         └───────────┐
                 ▼                                ▼
        ┌─────────────────┐              ┌─────────────────┐
        │   PostgreSQL     │              │      Redis       │
        │ transactional    │              │ cache, job queue,│
        │ source of truth  │              │ session/rate-limit│
        └─────────────────┘              └─────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────────────┐
        │              Provider adapters                 │
        │  PlacesProvider   (Google Places / mock)        │
        │  PlanningProvider (Anthropic / OpenAI / mock)   │
        │  TransportProvider(directions/time-matrix / mock)│
        │  BookingRefProvider (manual entry only, MVP)    │
        │  PaymentProvider  (Razorpay / Stripe — post-MVP)│
        └───────────────────────────────────────────────┘
```

## 4. Why a modular monolith, not microservices

MVP scope does not justify network-boundary complexity. Instead:

- One Go binary (`cmd/api`), one Postgres database.
- Strict internal package boundaries by domain (`internal/domain/trip`,
  `internal/domain/user`, …). No domain package imports another domain's
  internals — they talk through service-layer interfaces.
- Each domain owns its own migrations namespace and repository.
- This keeps a clean seam to extract a service later (e.g. the AI planning
  pipeline) if load ever demands it, without a rewrite.

## 5. Domain boundaries (MVP)

| Domain        | Responsibility                                             |
|---------------|--------------------------------------------------------------|
| `identity`    | Users, auth, sessions, password/OTP, JWT issuance            |
| `trip`        | Trip entity, lifecycle (draft → planning → voting → booked → done) |
| `membership`  | Invitations, join requests, roles (owner/co-organizer/member) |
| `preference`  | Per-member travel preferences (budget, dates, interests, pace) |
| `place`       | Cached, normalized place/destination data from providers      |
| `planning`    | Orchestrates retrieval + LLM generation + validation          |
| `itinerary`   | Structured itinerary entity, versions, edits                  |
| `voting`      | Poll options tied to itinerary versions or discrete decisions |
| `booking`     | Booking *references* (external confirmation numbers/links)   |
| `tripbook`    | Read-optimized aggregate view assembled from the above        |

Post-MVP domains (not built now): `notification`, `analytics`, `admin`,
`guide-marketplace`, full `inventory` (flights/hotels).

## 6. Data flow: itinerary generation (core critical path)

```
POST /trips/{id}/itinerary/generate
   │
   ▼
planning.Service.Generate(ctx, tripID)
   │
   ├─ 1. load Trip + Memberships + Preferences (Postgres)
   ├─ 2. PlacesProvider.Search / Details for destination(s)   -> []Place
   ├─ 3. TransportProvider.Matrix(places)                      -> travel times
   ├─ 4. build PlanningContext{ trip, prefs, places, matrix }
   ├─ 5. PlanningProvider.GenerateItinerary(ctx, PlanningContext)
   │        -> LLM call constrained to itinerary.Schema (JSON Schema / tool use)
   │        -> returns itinerary.Draft (structured, unvalidated)
   ├─ 6. validator.Validate(draft, places, matrix, prefs)
   │        -> checks: place existence, opening hours, no time overlap,
   │           feasible travel time between consecutive stops,
   │           per-day and total budget ceilings
   │        -> returns ValidationResult{ ok, violations[], repaired? }
   ├─ 7. if violations exist and are auto-repairable -> repair deterministically
   │    else -> bounded re-prompt (max N retries) with violation feedback
   ├─ 8. persist itinerary.Version (immutable) + mark current
   └─ 9. return validated itinerary to client
```

Validation lives entirely in Go, has 100% unit test coverage of its rules,
and never calls the LLM. It is the actual gate, not a courtesy check.

## 7. Frontend architecture

- Next.js (App Router) + React + TypeScript.
- Mobile-first: single-column layouts, bottom-sheet patterns for
  invite/vote/edit actions, large touch targets.
- Server components for read-heavy pages (Trip Book, itinerary view);
  client components for interactive forms (create trip, vote, preferences).
- API access through a typed client generated from the OpenAPI contract
  (`docs/api-contract.md` / `openapi.yaml`), not ad-hoc `fetch` calls.
- Auth: httpOnly, secure, SameSite=strict session cookie holding a short-lived
  JWT; refresh token rotation handled server-side. No token in
  `localStorage`.

## 8. Security posture

- Passwords: Argon2id (or bcrypt cost ≥ 12) — never stored plain, never logged.
- JWT: short-lived access token (15 min) + rotating refresh token, both
  server-issued httpOnly cookies. CSRF double-submit token for state-changing
  requests.
- AuthZ: every trip-scoped endpoint checks membership + role via middleware
  before hitting the handler; row-level checks in the repository layer as a
  second line of defense (defense in depth, not just middleware trust).
- Input validation: request DTOs validated with `go-playground/validator`
  at the handler boundary; domain invariants re-validated in the domain
  layer regardless of what the transport layer already checked.
- Secrets: never sent to the browser. All provider API keys live server-side
  in env vars / secret manager; the browser only ever talks to our own API.
- Rate limiting: Redis token-bucket per user/IP on auth and AI-generation
  endpoints (the LLM call is the expensive, abusable one).

## 9. Provider abstraction pattern

Every external dependency is an interface owned by the domain that uses it,
with a concrete adapter and a mock adapter, selected by config/env — never
by compile-time coupling to an SDK inside domain code.

```go
// internal/domain/place/provider.go
type Provider interface {
    Search(ctx context.Context, q SearchQuery) ([]Place, error)
    Details(ctx context.Context, placeID string) (Place, error)
}
```

```
internal/provider/place/
    google/       // real adapter, isolated SDK usage
    mock/         // deterministic fixture-based adapter
```

Selected via `PLACE_PROVIDER=google|mock` at boot. The mock adapter is not a
throwaway — it is a first-class, checked-in fixture set used in both local
dev and automated tests, so behavior is reproducible without live API calls
or keys.

## 10. What we explicitly do NOT build in MVP

- No hotel/flight inventory or live pricing engine — bookings are
  reference-only (user pastes a confirmation number / link from whatever
  they booked externally).
- No payments/split-expense engine (post-MVP).
- No guide marketplace.
- No microservices, message broker, or Kubernetes — a single Go binary,
  Postgres, Redis, deployed as a couple of containers is sufficient at MVP
  scale.
- No scraping — all third-party data comes through official provider APIs
  under their terms, via the provider adapter interfaces above.

## 11. Directory layout

```
trip-os/
  architecture.md
  docs/
    database-schema.sql
    api-contract.md
    implementation-plan.md
    environment-variables.md
    test-strategy.md
  backend/
    cmd/api/main.go
    internal/
      config/
      domain/{user,trip,membership,preference,place,planning,itinerary,voting,booking,tripbook}/
      service/
      handler/
      middleware/
      provider/{place,planning,transport,booking}/{<impl>,mock}/
      repository/postgres/
      validator/
      platform/{db,redis,logger,jwt}/
    migrations/
    Makefile
    .env.example
  frontend/
    app/
    components/
    lib/api-client/
  docker-compose.yml
```
