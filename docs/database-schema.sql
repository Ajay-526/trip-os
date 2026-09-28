-- Trip OS — MVP database schema
-- PostgreSQL 15+. Naming: snake_case, singular schema/plural table.
-- All PKs are UUIDv7-ish (generated in app layer) stored as uuid.
-- Timestamps are timestamptz, always UTC.

create extension if not exists pgcrypto;

-- =========================================================================
-- IDENTITY
-- =========================================================================

create table users (
    id                  uuid primary key default gen_random_uuid(),
    email               citext not null unique,
    phone               text unique,
    display_name        text not null,
    password_hash       text not null,
    email_verified_at   timestamptz,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now(),
    deleted_at          timestamptz
);

create table refresh_tokens (
    id              uuid primary key default gen_random_uuid(),
    user_id         uuid not null references users(id) on delete cascade,
    token_hash      text not null unique,
    issued_at       timestamptz not null default now(),
    expires_at      timestamptz not null,
    revoked_at      timestamptz,
    replaced_by     uuid references refresh_tokens(id)
);
create index idx_refresh_tokens_user on refresh_tokens(user_id);

-- =========================================================================
-- TRIP + MEMBERSHIP
-- =========================================================================

create type trip_status as enum (
    'draft', 'planning', 'voting', 'booked', 'in_progress', 'completed', 'cancelled'
);

create table trips (
    id                  uuid primary key default gen_random_uuid(),
    name                text not null,
    destination_query   text not null,          -- free-text user input, e.g. "Manali"
    start_date          date,
    end_date            date,
    budget_total_paise  bigint,                  -- money stored as integer paise
    currency            text not null default 'INR',
    status              trip_status not null default 'draft',
    created_by          uuid not null references users(id),
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now(),
    deleted_at          timestamptz,
    constraint chk_trip_dates check (end_date is null or start_date is null or end_date >= start_date)
);
create index idx_trips_created_by on trips(created_by);

create type membership_role as enum ('owner', 'co_organizer', 'member');
create type membership_status as enum ('invited', 'joined', 'declined', 'removed');

create table trip_memberships (
    id              uuid primary key default gen_random_uuid(),
    trip_id         uuid not null references trips(id) on delete cascade,
    user_id         uuid references users(id) on delete cascade, -- null until invite accepted by a real account
    invited_email   citext,                                       -- for invites sent before signup
    role            membership_role not null default 'member',
    status          membership_status not null default 'invited',
    invited_by      uuid references users(id),
    invited_at      timestamptz not null default now(),
    joined_at       timestamptz,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    constraint chk_membership_identity check (user_id is not null or invited_email is not null),
    constraint uq_membership_user unique (trip_id, user_id),
    constraint uq_membership_invite unique (trip_id, invited_email)
);
create index idx_memberships_trip on trip_memberships(trip_id);
create index idx_memberships_user on trip_memberships(user_id);

create table trip_invite_links (
    id              uuid primary key default gen_random_uuid(),
    trip_id         uuid not null references trips(id) on delete cascade,
    token_hash      text not null unique,
    created_by      uuid not null references users(id),
    max_uses        integer,
    use_count       integer not null default 0,
    expires_at      timestamptz,
    revoked_at      timestamptz,
    created_at      timestamptz not null default now()
);
create index idx_invite_links_trip on trip_invite_links(trip_id);

-- =========================================================================
-- PREFERENCES
-- =========================================================================

create table trip_member_preferences (
    id                  uuid primary key default gen_random_uuid(),
    trip_id             uuid not null references trips(id) on delete cascade,
    user_id             uuid not null references users(id) on delete cascade,
    budget_paise        bigint,                  -- individual budget ceiling
    pace                text check (pace in ('relaxed','balanced','packed')),
    interests           text[] not null default '{}',   -- e.g. {trekking, food, nightlife}
    dietary             text[] not null default '{}',
    mobility_notes      text,
    earliest_start_time time,
    latest_end_time     time,
    date_flexibility    text check (date_flexibility in ('fixed','plus_minus_2','flexible')),
    notes               text,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now(),
    constraint uq_pref_member unique (trip_id, user_id)
);
create index idx_prefs_trip on trip_member_preferences(trip_id);

-- =========================================================================
-- PLACE CACHE (normalized provider data, provider-agnostic)
-- =========================================================================

create table places (
    id                  uuid primary key default gen_random_uuid(),
    provider            text not null,           -- 'google' | 'mock' | ...
    provider_place_id   text not null,
    name                text not null,
    category            text,                    -- 'attraction' | 'restaurant' | 'stay' | ...
    lat                 double precision not null,
    lng                 double precision not null,
    address             text,
    opening_hours_json  jsonb,                   -- normalized weekly hours
    avg_visit_minutes   integer,
    price_level         smallint,                -- 0-4, provider scale
    rating              numeric(2,1),
    raw_json            jsonb,                    -- full provider payload, for audit/debug
    fetched_at          timestamptz not null default now(),
    constraint uq_place_provider unique (provider, provider_place_id)
);
-- Note: a proper geospatial index (earthdistance/cube, or PostGIS) is
-- deferred to Phase 1 when place-radius search is actually implemented.
-- earthdistance/cube require superuser to install on most managed Postgres
-- (including Render's free tier), so we avoid taking that dependency until
-- it's load-bearing. A plain btree pair is enough for exact/bounding-box
-- lookups in the meantime.
create index idx_places_lat_lng on places (lat, lng);

-- =========================================================================
-- PLANNING / ITINERARY
-- =========================================================================

create type itinerary_status as enum ('draft', 'validated', 'rejected', 'published', 'superseded');

create table itinerary_versions (
    id                  uuid primary key default gen_random_uuid(),
    trip_id             uuid not null references trips(id) on delete cascade,
    version_number      integer not null,
    status              itinerary_status not null default 'draft',
    generated_by        text not null,           -- 'ai' | 'manual_edit'
    planning_provider   text,                    -- e.g. 'anthropic:claude-sonnet-4-6'
    context_snapshot    jsonb not null,           -- retrieval context sent to LLM (for audit/repro)
    raw_model_output    jsonb,                    -- unvalidated LLM output, kept for audit
    validation_report   jsonb,                    -- violations found + repairs applied
    is_current          boolean not null default false,
    created_by          uuid references users(id),
    created_at          timestamptz not null default now(),
    constraint uq_itinerary_version unique (trip_id, version_number)
);
create unique index uq_itinerary_current on itinerary_versions(trip_id) where is_current;

create table itinerary_days (
    id                  uuid primary key default gen_random_uuid(),
    itinerary_version_id uuid not null references itinerary_versions(id) on delete cascade,
    day_number          integer not null,
    date                date,
    notes               text,
    constraint uq_itinerary_day unique (itinerary_version_id, day_number)
);

create table itinerary_stops (
    id                  uuid primary key default gen_random_uuid(),
    itinerary_day_id    uuid not null references itinerary_days(id) on delete cascade,
    place_id            uuid references places(id),
    sequence_index      integer not null,
    start_time          time not null,
    end_time            time not null,
    est_cost_paise      bigint not null default 0,
    travel_minutes_from_prev integer,
    notes               text,
    constraint chk_stop_time check (end_time > start_time),
    constraint uq_stop_sequence unique (itinerary_day_id, sequence_index)
);
create index idx_stops_day on itinerary_stops(itinerary_day_id);

-- =========================================================================
-- VOTING
-- =========================================================================

create type poll_status as enum ('open', 'closed');

create table trip_polls (
    id              uuid primary key default gen_random_uuid(),
    trip_id         uuid not null references trips(id) on delete cascade,
    title           text not null,
    description     text,
    poll_type       text not null check (poll_type in ('itinerary_choice','generic_choice')),
    status          poll_status not null default 'open',
    closes_at       timestamptz,
    created_by      uuid not null references users(id),
    created_at      timestamptz not null default now()
);
create index idx_polls_trip on trip_polls(trip_id);

create table trip_poll_options (
    id              uuid primary key default gen_random_uuid(),
    poll_id         uuid not null references trip_polls(id) on delete cascade,
    label           text not null,
    itinerary_version_id uuid references itinerary_versions(id),
    display_order   integer not null default 0
);

create table trip_poll_votes (
    id              uuid primary key default gen_random_uuid(),
    poll_id         uuid not null references trip_polls(id) on delete cascade,
    option_id       uuid not null references trip_poll_options(id) on delete cascade,
    user_id         uuid not null references users(id) on delete cascade,
    created_at      timestamptz not null default now(),
    constraint uq_vote_per_user unique (poll_id, user_id)
);

-- =========================================================================
-- BOOKING REFERENCES (no live inventory/payments in MVP)
-- =========================================================================

create type booking_category as enum ('stay','transport','activity','other');

create table booking_references (
    id                  uuid primary key default gen_random_uuid(),
    trip_id             uuid not null references trips(id) on delete cascade,
    category            booking_category not null,
    vendor_name         text not null,
    confirmation_code   text,
    external_url        text,
    cost_paise          bigint,
    currency            text not null default 'INR',
    attached_stop_id    uuid references itinerary_stops(id),
    added_by            uuid not null references users(id),
    notes               text,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);
create index idx_bookings_trip on booking_references(trip_id);

-- =========================================================================
-- AUDIT (lightweight, expand post-MVP)
-- =========================================================================

create table audit_log (
    id              bigserial primary key,
    trip_id         uuid references trips(id) on delete set null,
    actor_user_id   uuid references users(id) on delete set null,
    action          text not null,
    entity_type     text not null,
    entity_id       uuid,
    metadata        jsonb,
    created_at      timestamptz not null default now()
);
create index idx_audit_trip on audit_log(trip_id, created_at desc);
