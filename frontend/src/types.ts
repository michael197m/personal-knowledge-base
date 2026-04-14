export type Note = {
  id: string;
  title: string;
  content: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
};

export type HealthResponse = {
  status: string;
  service: string;
  timestamp: string;
  database: "ok" | "unavailable";
  embeddings: {
    status: "ok" | "unavailable" | "unconfigured";
    model?: string;
  };
};

export type CreateNoteInput = {
  title: string;
  content: string;
  tags: string[];
};

export type UpdateNoteInput = CreateNoteInput;

export type AuthResponse = {
  token: string;
  user: {
    id: string;
    email: string;
  };
};

export type SearchResult = {
  note: Note;
  matchType: "list" | "semantic" | "text";
  similarity?: number;
};
