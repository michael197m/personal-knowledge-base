import type {
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
  const response = await fetch(`${API_BASE_URL}/notes`);

  if (!response.ok) {
    throw new Error("Unable to fetch notes.");
  }

  const payload = (await response.json()) as Note[] | null;
  return payload ?? [];
}

export async function searchNotes(query: string): Promise<SearchResult[]> {
  const response = await fetch(
    `${API_BASE_URL}/search?q=${encodeURIComponent(query)}`,
  );

  if (!response.ok) {
    throw new Error("Unable to search notes.");
  }

  const payload = (await response.json()) as SearchResult[] | null;
  return payload ?? [];
}

export async function createNote(input: CreateNoteInput): Promise<Note> {
  const response = await fetch(`${API_BASE_URL}/notes`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
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
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error("Unable to update note.");
  }

  return response.json() as Promise<Note>;
}

export async function deleteNote(noteId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/notes/${noteId}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    throw new Error("Unable to delete note.");
  }
}
