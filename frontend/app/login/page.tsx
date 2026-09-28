"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState, type FormEvent } from "react";
import { api, ApiError, resetCsrfCache } from "@/lib/api-client/client";
import type { PublicUser } from "@/lib/api-client/types";

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="page empty-state"><span className="spinner" /></div>}>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const justSignedUp = searchParams.get("justSignedUp") === "1";
  const next = searchParams.get("next");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      resetCsrfCache();
      await api.post<{ user: PublicUser }>("/auth/login", { email, password }, { needsCsrf: false, noRefresh: true });
      router.push(next || "/trips");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not sign you in.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="page">
      <h1 style={{ fontSize: 24, marginBottom: 20 }}>Sign in</h1>
      {justSignedUp && (
        <div className="card" style={{ borderColor: "var(--color-success)" }}>
          Account created — sign in to continue.
        </div>
      )}
      {error && <div className="error-banner">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-field">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />
        </div>
        <div className="form-field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button type="submit" className="primary" disabled={submitting}>
          {submitting ? "Signing in…" : "Sign in"}
        </button>
      </form>
      <p className="muted" style={{ marginTop: 16 }}>
        New here? <Link href="/signup">Create an account</Link>
      </p>
    </div>
  );
}
