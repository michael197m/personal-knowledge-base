import { useEffect, useState } from "react";
import { NoteForm } from "./components/NoteForm";
import { NoteList } from "./components/NoteList";
import {
  createNote,
  deleteNote,
  fetchHealth,
  fetchNotes,
  login,
  register,
  searchNotes,
  setAuthToken,
  updateNote,
} from "./lib/api";
import type {
  CreateNoteInput,
  HealthResponse,
  Note,
  SearchResult,
  UpdateNoteInput,
} from "./types";

const seedHighlights = [
  "Semantic search over your notes",
  "Tags and rich note organization",
  "Local embeddings with Ollama",
];

function statusClass(status: "ok" | "unavailable" | "unconfigured"): string {
  if (status === "ok") {
    return "ok";
  }
  if (status === "unconfigured") {
    return "warn";
  }

  return "bad";
}

export default function App() {
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const [authEmail, setAuthEmail] = useState("");
  const [authPassword, setAuthPassword] = useState("");
  const [authToken, setLocalAuthToken] = useState<string | null>(null);
  const [authenticating, setAuthenticating] = useState(false);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [notes, setNotes] = useState<Note[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [editingNote, setEditingNote] = useState<Note | null>(null);
  const [deletingNoteId, setDeletingNoteId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchMode, setSearchMode] = useState(false);
  const [searchResults, setSearchResults] = useState<SearchResult[] | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const existingToken = localStorage.getItem("pkb_auth_token");
    if (existingToken) {
      setLocalAuthToken(existingToken);
      setAuthToken(existingToken);
    }

    async function load() {
      try {
        const [healthResponse, notesResponse] = await Promise.all([
          fetchHealth(),
          existingToken ? fetchNotes() : Promise.resolve([]),
        ]);
        setHealth(healthResponse);
        setNotes(notesResponse);
        setSearchResults(null);
      } catch (loadError) {
        setError(
          loadError instanceof Error ? loadError.message : "Unknown error",
        );
      }
    }

    void load();
  }, []);

  async function handleAuthSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setAuthenticating(true);

    try {
      const response =
        authMode === "register"
          ? await register(authEmail, authPassword)
          : await login(authEmail, authPassword);
      setAuthToken(response.token);
      setLocalAuthToken(response.token);
      localStorage.setItem("pkb_auth_token", response.token);
      const nextNotes = await fetchNotes();
      setNotes(nextNotes);
      setAuthPassword("");
    } catch (authError) {
      setError(authError instanceof Error ? authError.message : "Unknown error");
    } finally {
      setAuthenticating(false);
    }
  }

  function handleLogout() {
    setAuthToken(null);
    setLocalAuthToken(null);
    localStorage.removeItem("pkb_auth_token");
    setNotes([]);
    setSearchResults(null);
    setSearchMode(false);
    setEditingNote(null);
  }

  async function handleSubmit(input: CreateNoteInput) {
    if (!authToken) {
      setError("Please login first.");
      return;
    }

    setError(null);
    setSubmitting(true);

    try {
      const createdNote = await createNote(input);
      setNotes((currentNotes) => [createdNote, ...currentNotes]);
      setSearchMode(false);
      setSearchResults(null);
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : "Unknown error",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function handleUpdate(input: UpdateNoteInput) {
    if (!authToken) {
      setError("Please login first.");
      return;
    }

    if (!editingNote) {
      return;
    }

    setError(null);
    setSubmitting(true);

    try {
      const updatedNote = await updateNote(editingNote.id, input);
      setNotes((currentNotes) =>
        currentNotes.map((note) =>
          note.id === updatedNote.id ? updatedNote : note,
        ),
      );
      setSearchResults((currentResults) =>
        currentResults?.map((result) =>
          result.note.id === updatedNote.id
            ? { ...result, note: updatedNote }
            : result,
        ) ?? null,
      );
      setEditingNote(null);
    } catch (updateError) {
      setError(
        updateError instanceof Error ? updateError.message : "Unknown error",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(noteId: string) {
    if (!authToken) {
      setError("Please login first.");
      return;
    }

    setError(null);
    setDeletingNoteId(noteId);

    try {
      await deleteNote(noteId);
      setNotes((currentNotes) =>
        currentNotes.filter((note) => note.id !== noteId),
      );
      setSearchResults((currentResults) =>
        currentResults?.filter((result) => result.note.id !== noteId) ?? null,
      );
      setEditingNote((currentEditingNote) =>
        currentEditingNote?.id === noteId ? null : currentEditingNote,
      );
    } catch (deleteError) {
      setError(
        deleteError instanceof Error ? deleteError.message : "Unknown error",
      );
    } finally {
      setDeletingNoteId(null);
    }
  }

  async function handleSearchSubmit(event: React.FormEvent<HTMLFormElement>) {
    if (!authToken) {
      setError("Please login first.");
      return;
    }

    event.preventDefault();
    setError(null);

    try {
      const trimmedQuery = searchQuery.trim();
      if (trimmedQuery) {
        const nextResults = await searchNotes(trimmedQuery);
        setSearchResults(nextResults);
        setNotes(nextResults.map((result) => result.note));
      } else {
        const nextNotes = await fetchNotes();
        setSearchResults(null);
        setNotes(nextNotes);
      }
      setSearchMode(trimmedQuery.length > 0);
    } catch (searchError) {
      setError(
        searchError instanceof Error ? searchError.message : "Unknown error",
      );
    }
  }

  async function handleSearchClear() {
    if (!authToken) {
      setError("Please login first.");
      return;
    }

    setSearchQuery("");
    setSearchMode(false);
    setError(null);

    try {
      const nextNotes = await fetchNotes();
      setSearchResults(null);
      setNotes(nextNotes);
    } catch (loadError) {
      setError(
        loadError instanceof Error ? loadError.message : "Unknown error",
      );
    }
  }

  return (
    <main className="app-shell">
      <section className="hero">
        <p className="eyebrow">Personal Knowledge Base</p>
        <h1>Build a local-first note system with semantic search.</h1>
        <p className="hero-copy">
          This starter UI gives you a clean place to grow note editing, tag
          management, JWT auth, and vector search results without reworking the
          project structure later.
        </p>

        <div className="chip-row">
          {seedHighlights.map((item) => (
            <span key={item} className="chip">
              {item}
            </span>
          ))}
        </div>

        <form className="search-bar" onSubmit={(event) => void handleSearchSubmit(event)}>
          <input
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder="Search notes by title, content, or tag"
            disabled={!authToken}
          />
          <button type="submit" className="primary-button">
            Search
          </button>
          <button
            type="button"
            className="ghost-button"
            onClick={() => void handleSearchClear()}
            disabled={!authToken}
          >
            Clear
          </button>
        </form>
        {authToken ? (
          <button type="button" className="ghost-button" onClick={handleLogout}>
            Logout
          </button>
        ) : (
          <form className="auth-form" onSubmit={(event) => void handleAuthSubmit(event)}>
            <input
              type="email"
              placeholder="Email"
              value={authEmail}
              onChange={(event) => setAuthEmail(event.target.value)}
              required
            />
            <input
              type="password"
              placeholder="Password (min 8 chars)"
              value={authPassword}
              onChange={(event) => setAuthPassword(event.target.value)}
              minLength={8}
              required
            />
            <button type="submit" className="primary-button" disabled={authenticating}>
              {authenticating ? "Working..." : authMode === "register" ? "Create account" : "Login"}
            </button>
            <button
              type="button"
              className="ghost-button"
              onClick={() => setAuthMode((current) => (current === "login" ? "register" : "login"))}
              disabled={authenticating}
            >
              {authMode === "login" ? "Need an account?" : "Have an account?"}
            </button>
          </form>
        )}
      </section>

      <section className="status-grid">
        <article className="card">
          <h2>Backend</h2>
          {health ? (
            <>
              <p className="status ok">{health.status}</p>
              <p>{health.service}</p>
              <p className={`status ${authToken ? "ok" : "warn"}`}>
                Auth: {authToken ? "authenticated" : "not authenticated"}
              </p>
              <p className={`status ${statusClass(health.database)}`}>
                Database: {health.database}
              </p>
              <p className={`status ${statusClass(health.embeddings.status)}`}>
                Embeddings: {health.embeddings.status}
                {health.embeddings.model ? ` (${health.embeddings.model})` : ""}
              </p>
              <p className="muted">{health.timestamp}</p>
            </>
          ) : (
            <p className="muted">Waiting for API response...</p>
          )}
        </article>

        <article className="card">
          <h2>Notes</h2>
          <p className="status">{notes.length} persisted items</p>
          {!authToken ? (
            <p className="muted">
              Login or create an account to access your own notes.
            </p>
          ) : null}
          {searchMode ? (
            <>
              <p className="search-summary">
                Ranked search results for <strong>{searchQuery.trim()}</strong>
              </p>
              <p className="muted">
                Results prefer semantic vector similarity when embeddings exist
                and fall back to text matching otherwise.
              </p>
            </>
          ) : (
            <p className="muted">
              Notes now come from PostgreSQL. The next step is semantic search
              tuning and result explanation.
            </p>
          )}
        </article>
      </section>

      <section className="workspace">
        <article className="panel">
          <div className="panel-header">
            <h2>{editingNote ? "Edit note" : "Create note"}</h2>
            <span className="panel-badge">Write</span>
          </div>
          {editingNote
            ? (
                <NoteForm
                  heading="Update the selected note and replace its tags."
                  initialValue={{
                    title: editingNote.title,
                    content: editingNote.content,
                    tags: editingNote.tags,
                  }}
                  onSubmit={handleUpdate}
                  resetKey={editingNote.id}
                  submitLabel="Update note"
                  submitting={submitting}
                />
              )
            : (
                <NoteForm
                  heading="Create a note and persist it to PostgreSQL."
                  onSubmit={handleSubmit}
                  submitLabel="Save note"
                  submitting={submitting}
                />
              )}
          {!authToken ? <p className="muted">Authentication is required to create notes.</p> : null}
          {editingNote ? (
            <button
              type="button"
              className="ghost-button"
              onClick={() => setEditingNote(null)}
            >
              Cancel editing
            </button>
          ) : null}
        </article>

        <article className="panel">
          <div className="panel-header">
            <h2>API preview</h2>
            <span className="panel-badge">Live</span>
          </div>

          {error ? (
            <p className="error">{error}</p>
          ) : (
            <NoteList
              deletingNoteId={deletingNoteId}
              editingNoteId={editingNote?.id ?? null}
              emptyMessage={
                searchMode
                  ? "No notes matched this search. Try a different phrase or clear the query."
                  : undefined
              }
              notes={notes}
              results={searchResults ?? undefined}
              onDelete={handleDelete}
              onEdit={setEditingNote}
            />
          )}
        </article>
      </section>
    </main>
  );
}
