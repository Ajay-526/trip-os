# Trip OS — Environment Variables

None of these are ever sent to the browser. The Next.js app only ever calls
`NEXT_PUBLIC_API_BASE_URL`; every provider key lives on the Go backend.

## Backend (`backend/.env`)

| Variable | Required | Example | Notes |
|---|---|---|---|
| `APP_ENV` | yes | `development` \| `staging` \| `production` | |
| `HTTP_PORT` | yes | `8080` | |
| `DATABASE_URL` | yes | `postgres://trip_os:pass@localhost:5432/trip_os?sslmode=disable` | |
| `REDIS_URL` | yes | `redis://localhost:6379/0` | |
| `JWT_ACCESS_SECRET` | yes | (32+ byte random) | Signs short-lived access tokens |
| `JWT_ACCESS_TTL_MINUTES` | no | `15` | default 15 |
| `REFRESH_TOKEN_TTL_HOURS` | no | `720` | default 30 days |
| `COOKIE_DOMAIN` | yes (prod) | `.tripos.app` | Leave **unset** for a cross-domain deploy (e.g. Vercel frontend + Render backend) — let the browser default to the API's own host. |
| `COOKIE_SECURE` | yes (prod) | `true` | must be true outside local dev; forced to `true` automatically when `CROSS_SITE_COOKIES=true` |
| `CROSS_SITE_COOKIES` | no | `false` | Set to `true` when the frontend and backend are on different domains. Switches the session/refresh cookies to `SameSite=None` (required for the browser to send them cross-site) and requires `COOKIE_SECURE=true`. |
| `ALLOWED_ORIGIN` | required if `CROSS_SITE_COOKIES=true` | `https://trip-os.vercel.app` | Exact frontend origin allowed to make credentialed cross-origin requests (CORS). Must match exactly — no wildcards, no trailing slash. |
| `PORT` | no | `10000` | Most PaaS providers (Render, Railway) inject this automatically; takes priority over `HTTP_PORT` when set. |
| `CSRF_SECRET` | yes | (32+ byte random) | |
| `PASSWORD_HASH_MEMORY_KB` | no | `65536` | Argon2id tuning |
| `PASSWORD_HASH_ITERATIONS` | no | `3` | Argon2id tuning |
| `PLACE_PROVIDER` | yes | `mock` \| `google` | |
| `GOOGLE_PLACES_API_KEY` | only if `PLACE_PROVIDER=google` | | server-side only |
| `PLANNING_PROVIDER` | yes | `mock` \| `anthropic` | |
| `ANTHROPIC_API_KEY` | only if `PLANNING_PROVIDER=anthropic` | | server-side only |
| `ANTHROPIC_MODEL` | no | `claude-sonnet-4-6` | |
| `TRANSPORT_PROVIDER` | yes | `mock` \| `google` | directions/time-matrix |
| `RATE_LIMIT_AUTH_PER_MIN` | no | `10` | per-IP |
| `RATE_LIMIT_GENERATE_PER_HOUR` | no | `6` | per-user, the AI endpoint is the expensive one |
| `LOG_LEVEL` | no | `info` | |
| `SENTRY_DSN` | no | | error tracking, optional in dev |

## Frontend (`frontend/.env.local`)

| Variable | Required | Example | Notes |
|---|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | yes | `http://localhost:8080/api/v1` | only non-secret value the browser needs |

## Provider assumption notes

- **Places/Transport**: designed against the Google Places API + Routes API
  shape (place_id, opening_hours, geometry/location, distance matrix), but
  the domain interface (`place.Provider`, `transport.Provider`) is
  provider-neutral — any place/directions vendor can be adapted without
  touching domain or validation code. `PLACE_PROVIDER=mock` /
  `TRANSPORT_PROVIDER=mock` use checked-in fixtures and require no key,
  used by default in local dev and CI.
- **Planning/LLM**: designed against Anthropic's Messages API with
  structured/tool-constrained output. `PLANNING_PROVIDER=mock` returns a
  deterministic fixture itinerary and is the default for local dev/CI so
  the pipeline is testable without API cost or network access.
- **Bookings/Payments**: MVP has no live provider — `booking_references` are
  manually entered by users. No env vars needed until a payments/booking
  provider is introduced post-MVP.
