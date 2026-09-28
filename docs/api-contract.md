# Trip OS — API Contract (MVP)

Base URL: `/api/v1`. All bodies JSON. Auth via httpOnly session cookie
(`trip_os_session`); CSRF token required on mutating requests via
`X-CSRF-Token` header, issued at login and on `GET /auth/csrf`.

Conventions:
- IDs are UUIDs (strings).
- Money is integer paise + ISO currency code, never floats.
- Timestamps are ISO-8601 UTC.
- Errors: `{ "error": { "code": "STRING_CODE", "message": "human readable" } }`
  with appropriate HTTP status (400/401/403/404/409/422/429/500).
- Pagination: `?cursor=&limit=` request, `{ "items": [...], "next_cursor": "..." }` response.

## Auth

### POST /auth/signup
Request: `{ email, phone?, display_name, password }`
Response 201: `{ user: {id, email, display_name} }`
Errors: `409 EMAIL_TAKEN`, `422 VALIDATION_ERROR`

### POST /auth/login
Request: `{ email, password }`
Response 200: `{ user: {...} }` + sets session/refresh cookies
Errors: `401 INVALID_CREDENTIALS`, `429 RATE_LIMITED`

### POST /auth/logout
Response 204. Revokes refresh token, clears cookies.

### POST /auth/refresh
Rotates refresh token using the httpOnly refresh cookie. Response 200.

### GET /auth/me
Response 200: `{ user: {id, email, display_name} }` or 401.

## Trips

### POST /trips
Creates a trip; caller becomes `owner` membership automatically.
Request:
```json
{
  "name": "Manali Squad Trip",
  "destination_query": "Manali, Himachal Pradesh",
  "start_date": "2026-12-20",
  "end_date": "2026-12-24",
  "budget_total_paise": 3000000,
  "currency": "INR"
}
```
Response 201: `{ trip: TripDTO }`
Validation: `name` required 1-140 chars; dates optional but end >= start;
budget optional, must be >= 0 if present.

### GET /trips
List trips the caller is a member of. Response 200: `{ items: TripDTO[] }`

### GET /trips/{tripId}
Response 200: `{ trip: TripDTO, membership: MembershipDTO }` (caller's own
membership row, for role-aware UI). 403 if caller is not a member.

### PATCH /trips/{tripId}
Owner/co_organizer only. Partial update of name/dates/budget/status.
Response 200: `{ trip: TripDTO }`. 403 `NOT_AUTHORIZED` otherwise.

### DELETE /trips/{tripId}
Owner only. Soft delete. Response 204.

`TripDTO`:
```ts
{
  id: string; name: string; destination_query: string;
  start_date: string | null; end_date: string | null;
  budget_total_paise: number | null; currency: string;
  status: 'draft'|'planning'|'voting'|'booked'|'in_progress'|'completed'|'cancelled';
  member_count: number; created_by: string; created_at: string;
}
```

## Membership / Invitations

### POST /trips/{tripId}/invites
Owner/co_organizer only. Two modes:
- Direct email invite: `{ "email": "friend@x.com", "role": "member" }`
- Shareable link: `{ "link": true, "max_uses": 20, "expires_in_hours": 168 }`

Response 201 (email mode): `{ membership: MembershipDTO }`
Response 201 (link mode): `{ invite_link: { url, expires_at, max_uses } }`

### POST /invites/{token}/accept
Authenticated user accepts a link or email invite token.
Response 200: `{ membership: MembershipDTO }`
Errors: `410 INVITE_EXPIRED`, `409 INVITE_EXHAUSTED`, `404 INVITE_NOT_FOUND`

### GET /trips/{tripId}/members
Response 200: `{ items: MembershipDTO[] }`

### PATCH /trips/{tripId}/members/{membershipId}
Owner only: change role or status (`removed`).
Response 200: `{ membership: MembershipDTO }`

`MembershipDTO`:
```ts
{
  id: string; trip_id: string; user_id: string | null;
  invited_email: string | null; role: 'owner'|'co_organizer'|'member';
  status: 'invited'|'joined'|'declined'|'removed';
  joined_at: string | null;
}
```

## Preferences

### PUT /trips/{tripId}/preferences/me
Upsert caller's own preferences.
Request:
```json
{
  "budget_paise": 1500000,
  "pace": "balanced",
  "interests": ["trekking", "food", "photography"],
  "dietary": ["vegetarian"],
  "earliest_start_time": "08:00",
  "latest_end_time": "22:00",
  "date_flexibility": "plus_minus_2",
  "notes": "Prefer avoiding very high-altitude activities"
}
```
Response 200: `{ preference: PreferenceDTO }`

### GET /trips/{tripId}/preferences
Owner/co_organizer: all members' preferences (for planning context).
Members: only their own unless trip settings allow visibility (MVP: owner/co_organizer only).
Response 200: `{ items: PreferenceDTO[] }`

## Places (read-only, backed by provider + cache)

### GET /places/search?query=&near_lat=&near_lng=&category=
Response 200: `{ items: PlaceDTO[] }` (served from `places` cache table;
cache-miss triggers a provider call and upsert).

### GET /places/{placeId}
Response 200: `{ place: PlaceDTO }`

## Itinerary / Planning

### POST /trips/{tripId}/itinerary/generate
Owner/co_organizer only. Triggers the retrieve → ground → generate →
validate pipeline (see architecture.md §6). Synchronous for MVP (bounded by
timeout); returns the validated itinerary or a structured failure.

Response 200:
```json
{
  "itinerary_version": {
    "id": "...", "version_number": 2, "status": "validated",
    "days": [
      {
        "day_number": 1, "date": "2026-12-20",
        "stops": [
          {
            "place": { "id": "...", "name": "Solang Valley", "category": "attraction" },
            "start_time": "09:00", "end_time": "12:00",
            "est_cost_paise": 50000, "travel_minutes_from_prev": 0
          }
        ]
      }
    ],
    "validation_report": { "violations": [], "repairs": [] }
  }
}
```
Response 422 `ITINERARY_INFEASIBLE`: returned only if validation fails after
all bounded regeneration attempts; body includes `validation_report.violations`
so the UI can explain *why* (e.g. "budget exceeded by ₹4,200", "Rohtang Pass
closed on selected dates").

### GET /trips/{tripId}/itinerary
Returns current (`is_current = true`) validated version, or the latest draft
if none is current yet.
Response 200: `{ itinerary_version: ItineraryVersionDTO }`

### GET /trips/{tripId}/itinerary/versions
Response 200: `{ items: ItineraryVersionSummaryDTO[] }`

### PATCH /trips/{tripId}/itinerary/{versionId}/stops/{stopId}
Manual edit (owner/co_organizer). Re-runs the deterministic validator
(never the LLM) against the edited stop set before saving.
Response 200 on success; `422 VALIDATION_FAILED` with violations if the
manual edit breaks a hard rule (e.g. now overlaps another stop).

## Voting

### POST /trips/{tripId}/polls
Owner/co_organizer only.
Request: `{ title, description?, poll_type, options: [{label, itinerary_version_id?}], closes_at? }`
Response 201: `{ poll: PollDTO }`

### GET /trips/{tripId}/polls
Response 200: `{ items: PollDTO[] }` (includes live vote counts).

### POST /polls/{pollId}/vote
Request: `{ option_id }`. One vote per user per poll (upsert-replace).
Response 200: `{ poll: PollDTO }`

### POST /polls/{pollId}/close
Owner/co_organizer only. Response 200.

`PollDTO` includes `options: [{id, label, vote_count, itinerary_version_id}]`
and `my_vote: string | null`.

## Booking references

### POST /trips/{tripId}/bookings
Any joined member.
Request: `{ category, vendor_name, confirmation_code?, external_url?, cost_paise?, currency?, attached_stop_id?, notes? }`
Response 201: `{ booking: BookingRefDTO }`

Note: this endpoint records a *reference* the user asserts is true. Trip OS
never claims a booking is confirmed on the vendor's behalf — the UI must
visually distinguish "reference recorded" from "confirmed by vendor," and we
never fabricate or infer a confirmation state.

### GET /trips/{tripId}/bookings
Response 200: `{ items: BookingRefDTO[] }`

### DELETE /trips/{tripId}/bookings/{bookingId}
Owner/co_organizer or the member who added it.

## Trip Book (read-optimized aggregate)

### GET /trips/{tripId}/tripbook
Single call assembling everything a member needs on the day of/around the
trip: trip summary, current itinerary, member list, open polls, booking
references, grouped by day. Cached in Redis, invalidated on writes to any
underlying resource for that trip.

Response 200:
```json
{
  "trip": { "...TripDTO" },
  "members": [ "...MembershipDTO" ],
  "itinerary": { "...ItineraryVersionDTO" },
  "open_polls": [ "...PollDTO" ],
  "bookings_by_day": { "1": [ "...BookingRefDTO" ] }
}
```

## Status codes used across the API

`400` malformed request · `401` unauthenticated · `403` authenticated but
not authorized for this trip/role · `404` not found or not a member (we do
not leak existence of trips you're not on — `404`, not `403`, for that case)
· `409` conflict (duplicate vote, email already invited) · `410` gone
(expired invite) · `422` validation/business-rule failure · `429` rate
limited · `500` unexpected.
