const API_BASE = import.meta.env.VITE_API_URL ?? "/v1";

function getToken(): string | null {
  return localStorage.getItem("token");
}

export function setToken(token: string) {
  localStorage.setItem("token", token);
}

export function clearToken() {
  localStorage.removeItem("token");
}

export function isAuthenticated(): boolean {
  const token = getToken();
  if (!token) return false;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp * 1000 > Date.now();
  } catch {
    return false;
  }
}

export function getUser(): { id: string; email: string; name: string } | null {
  const token = getToken();
  if (!token) return null;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return { id: payload.sub, email: payload.email, name: payload.name };
  } catch {
    return null;
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });
  if (res.status === 401) {
    clearToken();
    window.location.href = "/login";
    throw new Error("Unauthorized");
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed: ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// --- Types ---

export interface Ingredient {
  id?: string;
  position: number;
  quantity?: string;
  unit?: string;
  name: string;
}

export interface Step {
  id?: string;
  position: number;
  instruction: string;
  timer_minutes?: number;
  timer_label?: string;
}

export interface Recipe {
  id: string;
  owner_id: string;
  title: string;
  description?: string;
  servings?: number;
  prep_time_mins?: number;
  cook_time_mins?: number;
  image_path?: string;
  source_url?: string;
  is_shared: boolean;
  tags: string[];
  ingredients: Ingredient[];
  steps: Step[];
  created_at: string;
  updated_at: string;
}

export interface RecipeInput {
  title: string;
  description?: string;
  servings?: number;
  prep_time_mins?: number;
  cook_time_mins?: number;
  source_url?: string;
  is_shared: boolean;
  tags: string[];
  ingredients: Omit<Ingredient, "id">[];
  steps: Omit<Step, "id">[];
}

// --- API calls ---

export const api = {
  recipes: {
    list: (params?: {
      q?: string;
      tag?: string;
      max_time?: number;
    }) => {
      const qs = new URLSearchParams();
      if (params?.q) qs.set("q", params.q);
      if (params?.tag) qs.set("tag", params.tag);
      if (params?.max_time) qs.set("max_time", String(params.max_time));
      return request<Recipe[]>(`/recipes?${qs}`);
    },
    get: (id: string) => request<Recipe>(`/recipes/${id}`),
    create: (data: RecipeInput) =>
      request<Recipe>("/recipes", {
        method: "POST",
        body: JSON.stringify(data),
      }),
    update: (id: string, data: RecipeInput) =>
      request<Recipe>(`/recipes/${id}`, {
        method: "PUT",
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      request<void>(`/recipes/${id}`, { method: "DELETE" }),
    uploadImage: (id: string, file: File) => {
      const form = new FormData();
      form.append("image", file);
      return request<{ image_path: string }>(`/recipes/${id}/image`, {
        method: "POST",
        body: form,
      });
    },
  },
  import: {
    fromURL: (url: string) =>
      request<RecipeInput>("/import/url", {
        method: "POST",
        body: JSON.stringify({ url }),
      }),
    fromPhoto: (file: File) => {
      const form = new FormData();
      form.append("image", file);
      return request<RecipeInput>("/import/photo", {
        method: "POST",
        body: form,
      });
    },
  },
  tags: {
    list: () => request<string[]>("/tags"),
  },
};
