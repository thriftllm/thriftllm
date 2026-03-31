// All API calls go to same-origin Next.js route handlers (/api/[...path])
// which proxy to the Go backend server-side. Auth is handled via HTTP-only cookies.

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  const res = await fetch(path, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errorData.error || errorData.message || `API error: ${res.status}`);
  }

  return res.json();
}

// ---- Setup ----
export async function getSetupStatus(): Promise<{ is_complete: boolean }> {
  return apiFetch("/api/setup/status");
}

export async function setupAdmin(data: { name: string; email: string; password: string }) {
  return apiFetch<{ token: string; user: any }>("/api/setup", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

// ---- Auth ----
export async function login(email: string, password: string) {
  return apiFetch<{ token: string; user: any }>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function logout() {
  return apiFetch("/api/auth/logout", { method: "POST" });
}

export async function getMe() {
  return apiFetch<any>("/api/auth/me");
}

// ---- Models ----
export async function listModels() {
  return apiFetch<any[]>("/api/models");
}

export async function createModel(data: any) {
  return apiFetch<any>("/api/models", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateModel(id: string, data: any) {
  return apiFetch<any>(`/api/models/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteModel(id: string) {
  return apiFetch(`/api/models/${id}`, { method: "DELETE" });
}

export async function toggleModel(id: string, active: boolean) {
  return apiFetch(`/api/models/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ is_active: active }),
  });
}

// ---- API Keys ----
export async function listAPIKeys() {
  return apiFetch<any[]>("/api/keys");
}

export async function createAPIKey(data: { name: string; rate_limit_rpm: number }) {
  return apiFetch<{ key: string; api_key: any }>("/api/keys", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteAPIKey(id: string) {
  return apiFetch(`/api/keys/${id}`, { method: "DELETE" });
}

export async function toggleAPIKey(id: string, active: boolean) {
  return apiFetch(`/api/keys/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ is_active: active }),
  });
}

// ---- Dashboard ----
export async function getDashboardOverview() {
  return apiFetch<any>("/api/dashboard/overview");
}

export async function getUsageData(range_: string = "7d") {
  return apiFetch<any[]>(`/api/dashboard/usage?range=${range_}`);
}

export async function getModelBreakdown(range_: string = "7d") {
  return apiFetch<any[]>(`/api/dashboard/models?range=${range_}`);
}

// ---- Requests ----
export async function listRequests(params: Record<string, string> = {}) {
  const query = new URLSearchParams(params).toString();
  return apiFetch<{ logs: any[]; total: number; page: number; limit: number; total_pages: number }>(
    `/api/requests?${query}`
  );
}

// ---- Cache ----
export async function getCacheStats() {
  return apiFetch<any>("/api/cache/stats");
}

export async function flushCache() {
  return apiFetch("/api/cache/flush", { method: "POST" });
}

// ---- Fallback Chains ----
export async function listFallbackChains() {
  return apiFetch<any[]>("/api/chains");
}

export async function createFallbackChain(data: {
  name: string;
  model_config_ids: string[];
  tag_selector?: string | null;
  is_default: boolean;
}) {
  return apiFetch<any>("/api/chains", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateFallbackChain(
  id: string,
  data: {
    name: string;
    model_config_ids: string[];
    tag_selector?: string | null;
    is_default: boolean;
  }
) {
  return apiFetch<any>(`/api/chains/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteFallbackChain(id: string) {
  return apiFetch(`/api/chains/${id}`, { method: "DELETE" });
}

// ---- Settings ----
export async function updateProfile(data: { name: string; email: string }) {
  return apiFetch("/api/settings/profile", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function changePassword(data: { current_password: string; new_password: string }) {
  return apiFetch("/api/settings/password", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}
