"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError } from "@/lib/api-client/client";
import { useCurrentUser } from "@/lib/api-client/useCurrentUser";
import type { Trip } from "@/lib/api-client/types";

export default function TripsPage() {
  const router = useRouter();
  const { user, loading: userLoading } = useCurrentUser();

  const [trips, setTrips] = useState<Trip[] | null>(null);
  const [listError, setListError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    if (userLoading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    refreshTrips();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userLoading, user]);

  function refreshTrips() {
    setListError(null);
    api
      .get<{ items: Trip[] }>("/trips")
      .then((res) => setTrips(res.items ?? []))
      .catch((err) => setListError(err instanceof ApiError ? err.message : "Could not load your trips."));
  }

  if (userLoading || !user) {
    return (
      <div className="page empty-state">
        <span className="spinner" />
      </div>
    );
  }

  return (
    <div className="page">
      <div className="appbar" style={{ margin: "-20px -16px 20px", padding: "14px 16px" }}>
        <span className="appbar-title">Your trips</span>
        <button className="primary" style={{ width: "auto" }} onClick={() => setShowCreate((v) => !v)}>
          {showCreate ? "Cancel" : "+ New trip"}
        </button>
      </div>

      {showCreate && (
        <CreateTripForm
          onCreated={(t) => {
            setShowCreate(false);
            router.push(`/trips/${t.id}`);
          }}
        />
      )}

      {listError && <div className="error-banner">{listError}</div>}

      {trips === null && !listError && (
        <div className="empty-state"><span className="spinner" /></div>
      )}

      {trips !== null && trips.length === 0 && (
        <div className="empty-state">
          <p>No trips yet.</p>
          <p className="muted">Start one and invite your squad.</p>
        </div>
      )}

      {trips?.map((t) => (
        <Link key={t.id} href={`/trips/${t.id}`}>
          <div className="card">
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <strong>{t.name}</strong>
              <span className="badge">{t.status.replace("_", " ")}</span>
            </div>
            <p className="muted" style={{ margin: "6px 0 0" }}>
              {t.destination_query}
              {t.start_date ? ` · ${t.start_date}${t.end_date ? ` → ${t.end_date}` : ""}` : ""}
            </p>
          </div>
        </Link>
      ))}
    </div>
  );
}

function CreateTripForm({ onCreated }: { onCreated: (t: Trip) => void }) {
  const [name, setName] = useState("");
  const [destination, setDestination] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [budget, setBudget] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const budgetPaise = budget.trim() ? Math.round(parseFloat(budget) * 100) : undefined;
      const res = await api.post<{ trip: Trip }>("/trips", {
        name,
        destination_query: destination,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
        budget_total_paise: budgetPaise,
        currency: "INR",
      });
      onCreated(res.trip);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create the trip.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="card">
      {error && <div className="error-banner">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-field">
          <label htmlFor="name">Trip name</label>
          <input id="name" required value={name} onChange={(e) => setName(e.target.value)} placeholder="Manali Squad Trip" />
        </div>
        <div className="form-field">
          <label htmlFor="destination">Destination</label>
          <input
            id="destination"
            required
            value={destination}
            onChange={(e) => setDestination(e.target.value)}
            placeholder="Manali, Himachal Pradesh"
          />
        </div>
        <div className="row">
          <div className="form-field">
            <label htmlFor="startDate">Start date</label>
            <input id="startDate" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
          </div>
          <div className="form-field">
            <label htmlFor="endDate">End date</label>
            <input id="endDate" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
          </div>
        </div>
        <div className="form-field">
          <label htmlFor="budget">Total budget (₹, optional)</label>
          <input
            id="budget"
            type="number"
            min="0"
            step="1"
            value={budget}
            onChange={(e) => setBudget(e.target.value)}
            placeholder="30000"
          />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? "Creating…" : "Create trip"}
        </button>
      </form>
    </div>
  );
}
