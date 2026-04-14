import type { Note, SearchResult } from "../types";

type NoteListProps = {
  deletingNoteId: string | null;
  editingNoteId: string | null;
  emptyMessage?: string;
  notes: Note[];
  results?: SearchResult[];
  onDelete: (noteId: string) => Promise<void>;
  onEdit: (note: Note) => void;
};

export function NoteList({
  deletingNoteId,
  editingNoteId,
  emptyMessage = "No notes yet. Create the first one to verify the full stack.",
  notes,
  results,
  onDelete,
  onEdit,
}: NoteListProps) {
  const displayResults =
    results ?? notes.map((note) => ({ matchType: "list" as const, note }));

  if (displayResults.length === 0) {
    return (
      <p className="muted">
        {emptyMessage}
      </p>
    );
  }

  return (
    <div className="notes-preview">
      {displayResults.map((result) => {
        const { note } = result;
        return (
        <article key={note.id} className="note-card">
          <div className="note-meta">
            <h3>{note.title}</h3>
            <span>{new Date(note.updatedAt).toLocaleString()}</span>
          </div>
          {result.matchType !== "list" ? (
            <div className="result-meta">
              <span className="result-badge">
                {result.matchType === "semantic" ? "Semantic match" : "Text fallback"}
              </span>
              {typeof result.similarity === "number" ? (
                <span className="result-score">
                  score {result.similarity.toFixed(2)}
                </span>
              ) : null}
            </div>
          ) : null}
          <p>{note.content}</p>
          <div className="note-actions">
            <button
              type="button"
              className="secondary-button"
              onClick={() => onEdit(note)}
            >
              {editingNoteId === note.id ? "Editing" : "Edit"}
            </button>
            <button
              type="button"
              className="danger-button"
              disabled={deletingNoteId === note.id}
              onClick={() => void onDelete(note.id)}
            >
              {deletingNoteId === note.id ? "Deleting..." : "Delete"}
            </button>
          </div>
          <div className="tag-row">
            {note.tags.map((tag) => (
              <span key={tag} className="tag">
                {tag}
              </span>
            ))}
          </div>
        </article>
        );
      })}
    </div>
  );
}
