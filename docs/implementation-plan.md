# Trip OS — Implementation Plan

Increments are intentionally small and sequenced so each one is independently
testable and shippable. "Done" for each increment = code + tests passing +
docs updated, per the project's stated rule for every feature.

## Phase 0 — Foundations (this delivery)
1. Repository & environment setup — Go module, Next.js app, docker-compose
   (Postgres + Redis), Makefile, `.env.example`, lint/format config.
2. Database schema & migrations — full MVP schema (see `database-schema.sql`),
   applied via a migration tool (`golang-migrate`), reversible.
3. Authentication & user model — signup/login/logout/refresh, Argon2id
   hashing, JWT access + refresh rotation, `/auth/me`, middleware.
4. Trip & membership domain — create/get/list/update/delete trip; owner
   membership auto-created; role/authorization middleware.
5. Trip creation UI — Next.js form with client + server validation,
   loading/error/empty states.
6. Invite/join workflow — email invite + shareable link, accept endpoint,
   join UI.
7. Group preference workflow — per-member preference upsert + aggregate
   view for organizers, preference form UI.

## Phase 1 — AI planning core (next delivery after Phase 0 is reviewed)
8. Destination/place provider abstraction — `place.Provider` interface,
   Google Places adapter, deterministic mock adapter with fixtures, Postgres
   cache repository.
9. AI planning provider abstraction — `planning.Provider` interface,
   Anthropic adapter using structured/tool-constrained output, mock adapter
   returning deterministic fixture itineraries.
10. Structured itinerary schema — JSON Schema for LLM output +
    corresponding Go structs, shared between generation and validation.
11. Itinerary validation engine — pure functions: place-existence check,
    opening-hours check, overlap check, travel-time feasibility check,
    budget check; bounded regeneration loop; 100% unit-tested with
    hand-built violation fixtures (this is the highest-risk correctness
    surface in the product and gets the most test investment).
12. Itinerary UI and editing — read view, manual stop edit that re-validates
    client-side optimistically and server-side authoritatively.

## Phase 2 — Group decision + memory
13. Group voting — poll creation tied to itinerary versions or generic
    decisions, live vote counts, close/finalize.
14. Booking-reference model and UI — manual reference entry, attached to a
    stop, clearly labeled as unconfirmed-by-us.
15. Trip Book — aggregate read view, Redis-cached, invalidated on writes.

## Phase 3 — Operational maturity
16. Notifications — invite sent, poll opened/closing soon, itinerary ready;
    starts as in-app + email, push deferred.
17. Analytics — trip funnel (created → invited → joined → planned → voted →
    booked), product usage; privacy-reviewed event schema before instrumenting.
18. Automated tests — expand from per-increment unit tests to end-to-end
    (Playwright) covering create → invite → join → preferences → generate →
    vote → book → trip book.
19. Admin foundation — internal read-only console for support/ops (trip
    lookup, provider call logs, feature flags). No destructive admin actions
    in MVP.
20. Production deployment and observability — containerized deploy,
    structured logging, tracing around the AI pipeline specifically
    (retrieval latency, generation latency, validation outcome distribution),
    error budgets/alerts on the generation endpoint.

## Sequencing rules

- No phase-N increment starts until phase-(N-1)'s increments compile, pass
  their tests, and their docs (env vars, API contract deltas) are updated.
- The AI planning pipeline (8–11) is built provider-agnostic and mock-first:
  the mock `PlacesProvider` and mock `PlanningProvider` are complete before
  any real provider key is required, so development and CI never depend on
  live third-party availability or cost.
- Validation (11) is written and tested against the schema (10) before the
  real LLM adapter (9) is wired to generate anything shown to a user —
  a passing validator with fixture data is the gate before "real" AI output
  is allowed to reach a trip.

## What's delivered in this response

Phase 0, items 1–7, implemented as real, compiling code (Go backend +
Next.js frontend skeleton), with the full Phase 0 schema included. Items 8–20
are specified above and scaffolded as interfaces/TODOs where useful, but not
implemented, per the instruction to build the MVP first rather than every
future feature at once.
