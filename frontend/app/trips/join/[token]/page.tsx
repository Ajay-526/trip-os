"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api-client/client";
import { useCurrentUser } from "@/lib/api-client/useCurrentUser";
import type { Membership } from "@/lib/api-client/types";

export default function JoinTripPage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const { user, loading: userLoading } = useCurrentUser();

  const [status, setStatus] = useState<"idle" | "joining" | "error">("idle");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (userLoading) return;
    if (!user) {
      // Preserve the invite token across sign-in by redirecting back here after login.
      router.replace(`/login?next=${encodeURIComponent(`/trips/join/${token}`)}`);
      return;
    }
    setStatus("joining");
    api
      .post<{ membership: Membership }>(`/invites/${token}/accept`)
      .then((res) => router.replace(`/trips/${res.membership.trip_id}`))
      .catch((err) => {
        setStatus("error");
        setError(
          err instanceof ApiError
            ? err.message
            : "This invite link could not be used. It may have expired."
        );
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userLoading, user, token]);

  return (
    <div className="page empty-state">
      {status !== "error" && <span className="spinner" />}
      {status === "error" && <div className="error-banner">{error}</div>}
    </div>
  );
}
