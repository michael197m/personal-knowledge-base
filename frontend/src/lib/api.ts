import type {
  AuthResponse,
  CreateNoteInput,
  HealthResponse,
  Note,
  SearchResult,
  UpdateNoteInput,
} from "../types";

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(`${API_BASE_URL}/health`);

  if (!response.ok) {
    throw new Error("Unable to fetch backend health.");
  }

  return response.json() as Promise<HealthResponse>;
}

export async function fetchNotes(): Promise<Note[]> {
  const response = await fetch(`${API_BASE_URL}/notes`, {
    credentials: "include",
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("authentication required");
    }
    throw new Error("Unable to fetch notes.");
  }

  const payload = (await response.json()) as Note[] | null;
  return payload ?? [];
}

export async function searchNotes(query: string): Promise<SearchResult[]> {
  const response = await fetch(
    `${API_BASE_URL}/search?q=${encodeURIComponent(query)}`,
    {
      credentials: "include",
    },
  );

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("authentication required");
    }
    throw new Error("Unable to search notes.");
  }

  const payload = (await response.json()) as SearchResult[] | null;
  return payload ?? [];
}

export async function createNote(input: CreateNoteInput): Promise<Note> {
  const response = await fetch(`${API_BASE_URL}/notes`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("authentication required");
    }
    throw new Error("Unable to create note.");
  }

  return response.json() as Promise<Note>;
}

export async function updateNote(
  noteId: string,
  input: UpdateNoteInput,
): Promise<Note> {
  const response = await fetch(`${API_BASE_URL}/notes/${noteId}`, {
    method: "PUT",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("authentication required");
    }
    throw new Error("Unable to update note.");
  }

  return response.json() as Promise<Note>;
}

export async function deleteNote(noteId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/notes/${noteId}`, {
    method: "DELETE",
    credentials: "include",
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("authentication required");
    }
    throw new Error("Unable to delete note.");
  }
}

export async function register(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await fetch(`${API_BASE_URL}/auth/register`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const payload = (await response.json()) as { error?: string };
    throw new Error(payload.error ?? "Unable to register.");
  }

  return response.json() as Promise<AuthResponse>;
}

export async function login(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const payload = (await response.json()) as { error?: string };
    throw new Error(payload.error ?? "Unable to login.");
  }

  return response.json() as Promise<AuthResponse>;
}

export async function logout(): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/auth/logout`, {
    method: "POST",
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Unable to logout.");
  }
}
