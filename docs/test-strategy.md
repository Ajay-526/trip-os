# Trip OS — Test Strategy

## Principles

- Domain and validation logic is unit-tested with no network/DB dependency.
- Repository logic is tested against a real Postgres (via `testcontainers`
  or a docker-compose test database), never mocked SQL — schema drift bugs
  are exactly what repository tests exist to catch.
- Provider adapters are tested against their mock implementation by default;
  the real adapters get a small "contract test" suite runnable manually
  against live credentials, excluded from CI.
- The itinerary validator gets the highest test investment in the codebase:
  it is the deterministic gate the whole "LLM is not the source of truth"
  guarantee rests on.

## Layers

### 1. Domain unit tests (`internal/domain/**/*_test.go`)
- Entity invariants (e.g. a trip's `end_date` cannot precede `start_date`;
  a membership cannot be `joined` without a `user_id`).
- Pure business rules with table-driven tests.
- Run on every commit, < 5s total.

### 2. Validator unit tests (`internal/validator/*_test.go`) — critical path
Fixture-driven, one test per rule, each with a passing and failing case:
- **Existence**: every referenced `place_id` in the LLM draft must exist in
  the retrieved place set. Fabricated/unknown place → violation.
- **Opening hours**: stop's `[start_time, end_time)` must fall inside the
  place's normalized opening hours for that weekday. Closed-day or
  out-of-hours stop → violation.
- **Overlap**: within a day, no two stops' time ranges may intersect, and
  stops must be in non-decreasing sequence order.
- **Travel feasibility**: `travel_minutes_from_prev` (from the transport
  matrix, not the LLM's claim) plus the previous stop's end time must be
  `<=` the next stop's start time. LLM-asserted travel time is never trusted
  — the validator recomputes it from the matrix.
- **Budget**: sum of `est_cost_paise` per day and in total must not exceed
  the trip/member budget ceilings, with a documented tolerance (e.g. 0%,
  hard ceiling — MVP treats budget as a hard constraint, not a soft one).
- Additional test: a draft with zero violations passes unchanged (no
  false positives) — as important as catching real violations.

### 3. Repository/integration tests (`internal/repository/postgres/*_test.go`)
- Run against a real ephemeral Postgres instance (docker-compose service or
  testcontainers), migrations applied fresh per test run.
- Cover: unique constraints (one membership per user per trip, one vote per
  user per poll), cascade deletes, the `is_current` partial-unique-index
  invariant on `itinerary_versions`.

### 4. Handler/API tests (`internal/handler/*_test.go`)
- `httptest` server, real router + middleware, repository backed by the same
  ephemeral Postgres as layer 3.
- Cover authz boundaries explicitly: a non-member gets `404` (not `403`) on
  a trip; a `member` gets `403` on an owner-only action; role escalation
  attempts are rejected.
- Cover input validation boundaries (missing required fields, malformed
  dates, negative budgets).

### 5. Provider mock adapters (`internal/provider/**/mock/*_test.go`)
- Assert the mock adapters satisfy the exact same interface contract as the
  real ones and return schema-valid, deterministic data — they are the
  backbone of layers 2–4 running without live keys.

### 6. End-to-end tests (post Phase 0 — Playwright, `e2e/`)
Full user journeys against a running stack (docker-compose: Postgres, Redis,
backend, frontend, all mock providers):
- Sign up → create trip → invite via link → second user joins → both submit
  preferences → generate itinerary (mock planning provider) → itinerary
  passes validation and renders → open a poll → both vote → add a booking
  reference → Trip Book reflects everything.
- A deliberately infeasible mock-planning fixture is used once to assert the
  `422 ITINERARY_INFEASIBLE` path renders a clear explanation in the UI
  rather than a broken itinerary.

### 7. Frontend component tests (`frontend/**/*.test.tsx`, Vitest + Testing
Library)
- Form validation (trip creation, preferences) — required fields, date
  ordering, budget non-negative.
- Loading/error/empty states render for every data-fetching component per
  the project's stated requirement — each list/detail component has an
  explicit test for its empty state and its error state, not just the
  happy path.

## CI gates

1. `go vet` + `staticcheck` + `golangci-lint`.
2. `go test ./...` (layers 1–5) against ephemeral Postgres service container.
3. `npm run lint` + `npm run typecheck` + `vitest run` (layer 7).
4. E2E suite (layer 6) runs on merge to main, not on every push (slower).
5. No merge if the validator package's coverage drops below its current
   baseline — enforced explicitly given its correctness-critical role.

## What is deliberately out of scope for automated tests in MVP

- Load/performance testing of the AI generation endpoint (revisit once real
  usage patterns exist).
- Visual regression testing of the frontend.
- Real-provider contract tests running in CI (cost/flakiness) — run manually
  before a provider version bump instead.
