import type { ApiErrorBody } from "./types";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

let cachedCsrfToken: string | null = null;

async function getCsrfToken(): Promise<string> {
  if (cachedCsrfToken) return cachedCsrfToken;
  const res = await fetch(`${API_BASE_URL}/auth/csrf`, { credentials: "include" });
  if (!res.ok) throw new ApiError(res.status, "CSRF_FETCH_FAILED", "could not fetch CSRF token");
  const body = (await res.json()) as { csrf_token: string };
  cachedCsrfToken = body.csrf_token;
  return cachedCsrfToken;
}

// The CSRF token is derived from the session cookie, so it must be dropped
// whenever the session changes (login, logout, token refresh).
export function resetCsrfCache(): void {
  cachedCsrfToken = null;
}

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE";
  body?: unknown;
  // Defaults to true for non-GET. Must be false for signup/login/refresh:
  // the caller has no session yet, and /auth/csrf requires one.
  needsCsrf?: boolean;
  // Skip the automatic refresh-and-retry on 401 (used by the auth endpoints).
  noRefresh?: boolean;
}

let refreshInFlight: Promise<boolean> | null = null;

// Access tokens live 15 minutes; the refresh cookie lasts much longer.
// Concurrent 401s share a single refresh call.
function tryRefresh(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = fetch(`${API_BASE_URL}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    })
      .then((res) => {
        if (res.ok) resetCsrfCache();
        return res.ok;
      })
      .catch(() => false)
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
}

async function send(path: string, opts: RequestOptions): Promise<Response> {
  const method = opts.method ?? "GET";
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (opts.needsCsrf ?? method !== "GET") {
    headers["X-CSRF-Token"] = await getCsrfToken();
  }
  return fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    credentials: "include",
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  let res: Response;
  try {
    res = await send(path, opts);
  } catch (err) {
    // The CSRF pre-fetch itself returns 401 when the access token has expired.
    if (err instanceof ApiError && err.status === 401 && !opts.noRefresh && (await tryRefresh())) {
      res = await send(path, opts);
    } else {
      throw err;
    }
  }

  if (res.status === 401 && !opts.noRefresh && (await tryRefresh())) {
    res = await send(path, opts);
  } else if (res.status === 403) {
    const body = (await res.clone().json().catch(() => null)) as ApiErrorBody | null;
    if (body?.error?.code === "CSRF_INVALID") {
      resetCsrfCache();
      res = await send(path, opts);
    }
  }

  if (res.status === 204) return undefined as T;

  const isJson = res.headers.get("content-type")?.includes("application/json");
  const payload = isJson ? await res.json() : undefined;

  if (!res.ok) {
    const errBody = payload as ApiErrorBody | undefined;
    throw new ApiError(
      res.status,
      errBody?.error?.code ?? "UNKNOWN_ERROR",
      errBody?.error?.message ?? "Something went wrong. Please try again."
    );
  }
  return payload as T;
}

type CallOpts = Pick<RequestOptions, "needsCsrf" | "noRefresh">;

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown, o?: CallOpts) => request<T>(path, { method: "POST", body, ...o }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: "PATCH", body }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: "PUT", body }),
  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
