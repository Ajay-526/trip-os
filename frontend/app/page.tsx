"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useCurrentUser } from "@/lib/api-client/useCurrentUser";

export default function HomePage() {
  const { user, loading } = useCurrentUser();
  const router = useRouter();

  useEffect(() => {
    if (!loading && user) {
      router.replace("/trips");
    }
  }, [loading, user, router]);

  if (loading) {
    return (
      <div className="page empty-state">
        <span className="spinner" />
      </div>
    );
  }

  if (user) {
    // Redirecting; render nothing to avoid a flash of the landing content.
    return null;
  }

  return (
    <div className="page">
      <h1 style={{ fontSize: 28, marginBottom: 6 }}>Trip OS</h1>
      <p className="muted" style={{ marginBottom: 28 }}>
        Plan group trips with your people — preferences in, AI-assisted itinerary out, everyone
        votes, one live Trip Book for the whole squad.
      </p>
      <div className="row" style={{ marginBottom: 12 }}>
        <Link href="/signup"><button className="primary">Create an account</button></Link>
      </div>
      <div className="row">
        <Link href="/login"><button className="secondary">Sign in</button></Link>
      </div>
    </div>
  );
}
