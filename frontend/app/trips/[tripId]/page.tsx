"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError } from "@/lib/api-client/client";
import { useCurrentUser } from "@/lib/api-client/useCurrentUser";
import type { Membership, MembershipRole, Preference, Trip } from "@/lib/api-client/types";

const INTEREST_OPTIONS = ["trekking", "food", "nightlife", "photography", "shopping", "relaxation", "culture", "adventure sports"];

export default function TripDetailPage() {
  const { tripId } = useParams<{ tripId: string }>();
  const router = useRouter();
  const { user, loading: userLoading } = useCurrentUser();

  const [trip, setTrip] = useState<Trip | null>(null);
  const [membership, setMembership] = useState<Membership | null>(null);
  const [members, setMembers] = useState<Membership[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (userLoading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userLoading, user, tripId]);

  function load() {
    setError(null);
    api
      .get<{ trip: Trip; membership: Membership }>(`/trips/${tripId}`)
      .then((res) => {
        setTrip(res.trip);
        setMembership(res.membership);
      })
      .catch((err) => {
        setError(
          err instanceof ApiError && err.status === 404
            ? "This trip doesn't exist, or you're not on it."
            : "Could not load this trip."
        );
      });
    api
      .get<{ items: Membership[] }>(`/trips/${tripId}/members`)
      .then((res) => setMembers(res.items ?? []))
      .catch(() => {
        /* members list is secondary; the page still works without it */
      });
  }

  if (userLoading || !user) {
    return <div className="page empty-state"><span className="spinner" /></div>;
  }

  if (error) {
    return (
      <div className="page">
        <div className="error-banner">{error}</div>
      </div>
    );
  }

  if (!trip || !membership) {
    return <div className="page empty-state"><span className="spinner" /></div>;
  }

  const canManage = membership.role === "owner" || membership.role === "co_organizer";

  return (
    <div className="page">
      <div className="appbar" style={{ margin: "-20px -16px 20px", padding: "14px 16px" }}>
        <span className="appbar-title">{trip.name}</span>
        <span className="badge">{trip.status.replace("_", " ")}</span>
      </div>

      <div className="card">
        <p className="muted" style={{ margin: 0 }}>{trip.destination_query}</p>
        {trip.start_date && (
          <p className="muted" style={{ margin: "4px 0 0" }}>
            {trip.start_date} → {trip.end_date ?? "?"}
          </p>
        )}
        {trip.budget_total_paise != null && (
          <p className="muted" style={{ margin: "4px 0 0" }}>
            Group budget: ₹{(trip.budget_total_paise / 100).toLocaleString("en-IN")}
          </p>
        )}
      </div>

      <MembersSection tripId={tripId} members={members} canManage={canManage} onChanged={load} />

      <PreferencesSection tripId={tripId} />

      <div className="card" style={{ opacity: 0.6 }}>
        <strong>Itinerary</strong>
        <p className="muted" style={{ margin: "6px 0 0" }}>
          AI-assisted itinerary generation ships in the next phase, once every
          member has set their preferences.
        </p>
      </div>
    </div>
  );
}

function MembersSection({
  tripId,
  members,
  canManage,
  onChanged,
}: {
  tripId: string;
  members: Membership[] | null;
  canManage: boolean;
  onChanged: () => void;
}) {
  const [showInvite, setShowInvite] = useState(false);

  return (
    <div className="card">
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <strong>Members{members ? ` (${members.length})` : ""}</strong>
        {canManage && (
          <button className="link" onClick={() => setShowInvite((v) => !v)}>
            {showInvite ? "Close" : "+ Invite"}
          </button>
        )}
      </div>

      {members === null && <p className="muted">Loading members…</p>}
      {members?.length === 0 && <p className="muted">No members yet.</p>}
      {members?.map((m) => (
        <div key={m.id} style={{ display: "flex", justifyContent: "space-between", padding: "8px 0", borderTop: "1px solid var(--color-border)" }}>
          <span>{m.invited_email ?? m.user_id ?? "Member"}</span>
          <span className={`badge ${m.role}`}>{m.role.replace("_", " ")}</span>
        </div>
      ))}

      {showInvite && (
        <InviteForm
          tripId={tripId}
          onDone={() => {
            setShowInvite(false);
            onChanged();
          }}
        />
      )}
    </div>
  );
}

function InviteForm({ tripId, onDone }: { tripId: string; onDone: () => void }) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<MembershipRole>("member");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [linkUrl, setLinkUrl] = useState<string | null>(null);

  async function inviteByEmail(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.post(`/trips/${tripId}/invites`, { email, role });
      onDone();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not send the invite.");
    } finally {
      setSubmitting(false);
    }
  }

  async function createLink() {
    setSubmitting(true);
    setError(null);
    try {
      const res = await api.post<{ invite_link: { url: string } }>(`/trips/${tripId}/invites`, {
        link: true,
        max_uses: 20,
        expires_in_hours: 168,
      });
      setLinkUrl(res.invite_link.url);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create an invite link.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div style={{ marginTop: 14, paddingTop: 14, borderTop: "1px solid var(--color-border)" }}>
      {error && <div className="error-banner">{error}</div>}
      {linkUrl && (
        <div className="card" style={{ marginBottom: 12, wordBreak: "break-all" }}>
          <p className="muted" style={{ margin: "0 0 6px" }}>Shareable link (expires in 7 days, up to 20 uses):</p>
          <code>{linkUrl}</code>
        </div>
      )}
      <form onSubmit={inviteByEmail}>
        <div className="row">
          <div className="form-field" style={{ flex: 2 }}>
            <label htmlFor="inviteEmail">Email</label>
            <input id="inviteEmail" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="friend@example.com" />
          </div>
          <div className="form-field">
            <label htmlFor="inviteRole">Role</label>
            <select id="inviteRole" value={role} onChange={(e) => setRole(e.target.value as MembershipRole)}>
              <option value="member">Member</option>
              <option value="co_organizer">Co-organizer</option>
            </select>
          </div>
        </div>
        <div className="row">
          <button type="submit" className="primary" disabled={submitting}>Send email invite</button>
          <button type="button" className="secondary" disabled={submitting} onClick={createLink}>Get shareable link</button>
        </div>
      </form>
    </div>
  );
}

function PreferencesSection({ tripId }: { tripId: string }) {
  const [pref, setPref] = useState<Preference | null | undefined>(undefined); // undefined = loading
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const [budget, setBudget] = useState("");
  const [pace, setPace] = useState<Preference["pace"]>("");
  const [interests, setInterests] = useState<string[]>([]);
  const [notes, setNotes] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    api
      .get<{ items: Preference[] }>(`/trips/${tripId}/preferences`)
      .then((res) => {
        const mine = res.items?.[0] ?? null;
        setPref(mine);
        if (mine) {
          setBudget(mine.budget_paise != null ? String(mine.budget_paise / 100) : "");
          setPace(mine.pace);
          setInterests(mine.interests ?? []);
          setNotes(mine.notes ?? "");
        }
      })
      .catch(() => setError("Could not load your preferences."));
  }, [tripId]);

  function toggleInterest(i: string) {
    setInterests((cur) => (cur.includes(i) ? cur.filter((x) => x !== i) : [...cur, i]));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setSaved(false);
    try {
      await api.put(`/trips/${tripId}/preferences/me`, {
        budget_paise: budget.trim() ? Math.round(parseFloat(budget) * 100) : undefined,
        pace: pace || undefined,
        interests,
        date_flexibility: undefined,
        notes: notes || undefined,
      });
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save your preferences.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="card">
      <strong>Your preferences</strong>
      <p className="muted" style={{ margin: "4px 0 14px" }}>
        This is what the AI planner uses when it builds the itinerary — the more
        specific, the better the plan.
      </p>

      {pref === undefined && <p className="muted">Loading…</p>}
      {error && <div className="error-banner">{error}</div>}

      {pref !== undefined && (
        <form onSubmit={handleSubmit}>
          <div className="form-field">
            <label htmlFor="myBudget">Your personal budget (₹, optional)</label>
            <input id="myBudget" type="number" min="0" value={budget} onChange={(e) => setBudget(e.target.value)} placeholder="15000" />
          </div>
          <div className="form-field">
            <label htmlFor="pace">Pace</label>
            <select id="pace" value={pace} onChange={(e) => setPace(e.target.value as Preference["pace"])}>
              <option value="">No preference</option>
              <option value="relaxed">Relaxed</option>
              <option value="balanced">Balanced</option>
              <option value="packed">Packed</option>
            </select>
          </div>
          <div className="form-field">
            <label>Interests</label>
            <div className="chip-row">
              {INTEREST_OPTIONS.map((i) => (
                <span key={i} className={`chip ${interests.includes(i) ? "selected" : ""}`} onClick={() => toggleInterest(i)}>
                  {i}
                </span>
              ))}
            </div>
          </div>
          <div className="form-field">
            <label htmlFor="notes">Anything else the group should know?</label>
            <textarea id="notes" rows={3} value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="e.g. avoid very high-altitude activities" />
          </div>
          <button type="submit" className="primary" disabled={submitting}>
            {submitting ? "Saving…" : "Save preferences"}
          </button>
          {saved && <p className="muted" style={{ marginTop: 8, color: "var(--color-success)" }}>Saved.</p>}
        </form>
      )}
    </div>
  );
}
