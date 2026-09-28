-- Reverses 0001_init.up.sql. Drops in dependency order (children first).

drop table if exists audit_log;
drop table if exists booking_references;
drop type if exists booking_category;

drop table if exists trip_poll_votes;
drop table if exists trip_poll_options;
drop table if exists trip_polls;
drop type if exists poll_status;

drop table if exists itinerary_stops;
drop table if exists itinerary_days;
drop table if exists itinerary_versions;
drop type if exists itinerary_status;

drop table if exists places;

drop table if exists trip_member_preferences;

drop table if exists trip_invite_links;
drop table if exists trip_memberships;
drop type if exists membership_status;
drop type if exists membership_role;

drop table if exists trips;
drop type if exists trip_status;

drop table if exists refresh_tokens;
drop table if exists users;

drop extension if exists citext;
drop extension if exists earthdistance;
drop extension if exists cube;
drop extension if exists pgcrypto;
