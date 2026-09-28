"use client";

import { useEffect, useState } from "react";
import { api, ApiError } from "./client";
import type { PublicUser } from "./types";

interface AuthState {
  user: PublicUser | null;
  loading: boolean;
  error: string | null;
}

// Auth truth is the httpOnly session cookie, checked server-side on every
// request. This hook only reflects that state for the UI (which screen to
// show); it never stores a token client-side.
export function useCurrentUser(): AuthState {
  const [state, setState] = useState<AuthState>({ user: null, loading: true, error: null });

  useEffect(() => {
    let cancelled = false;
    api
      .get<{ user: PublicUser }>("/auth/me")
      .then((res) => {
        if (!cancelled) setState({ user: res.user, loading: false, error: null });
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          setState({ user: null, loading: false, error: null });
        } else {
          setState({ user: null, loading: false, error: "Could not check your sign-in status." });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return state;
}
