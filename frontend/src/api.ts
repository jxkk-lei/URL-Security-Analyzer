export interface URLCheck {
  id: number;
  url: string;
  available: boolean;
  status_code?: number;
  response_time_ms: number;
  error_message?: string;
  checked_at: string;
}

const API_BASE_URL = "/api/v1";

async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message.trim() || `HTTP ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export function checkURL(url: string): Promise<URLCheck> {
  return request<URLCheck>("/checks", {
    method: "POST",
    body: JSON.stringify({ url }),
  });
}

export async function getChecks(limit = 20): Promise<URLCheck[]> {
  const checks = await request<URLCheck[]>(
    `/checks?limit=${limit}`,
  );

  return checks ?? [];
}

export function getHealth(): Promise<{ status: string }> {
  return request<{ status: string }>("/health");
}