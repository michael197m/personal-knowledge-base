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
};

export type CreateNoteInput = {
  title: string;
  content: string;
  tags: string[];
};

export type UpdateNoteInput = CreateNoteInput;

export type SearchResult = {
  note: Note;
  matchType: "list" | "semantic" | "text";
  similarity?: number;
};
