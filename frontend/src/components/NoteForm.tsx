import { useEffect, useState } from "react";
import type { CreateNoteInput } from "../types";

const initialFormState: CreateNoteInput = {
  title: "",
  content: "",
  tags: [],
};

type NoteFormProps = {
  onSubmit: (input: CreateNoteInput) => Promise<void>;
  initialValue?: CreateNoteInput;
  submitLabel?: string;
  heading?: string;
  resetKey?: string;
  submitting: boolean;
};

export function NoteForm({
  onSubmit,
  initialValue = initialFormState,
  submitLabel = "Save note",
  heading,
  resetKey,
  submitting,
}: NoteFormProps) {
  const [form, setForm] = useState(initialValue);
  const [tagInput, setTagInput] = useState(initialValue.tags.join(", "));

  useEffect(() => {
    setForm(initialValue);
    setTagInput(initialValue.tags.join(", "));
  }, [initialValue, resetKey]);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    await onSubmit({
      title: form.title,
      content: form.content,
      tags: tagInput
        .split(",")
        .map((tag) => tag.trim())
        .filter(Boolean),
    });

    setForm(initialFormState);
    setTagInput("");
  }

  return (
    <form className="note-form" onSubmit={(event) => void handleSubmit(event)}>
      {heading ? <p className="form-heading">{heading}</p> : null}
      <label className="field">
        <span>Title</span>
        <input
          required
          value={form.title}
          onChange={(event) =>
            setForm((current) => ({
              ...current,
              title: event.target.value,
            }))
          }
          placeholder="Semantic search ideas"
        />
      </label>

      <label className="field">
        <span>Content</span>
        <textarea
          required
          rows={7}
          value={form.content}
          onChange={(event) =>
            setForm((current) => ({
              ...current,
              content: event.target.value,
            }))
          }
          placeholder="Store embeddings on write and rank matches by cosine distance."
        />
      </label>

      <label className="field">
        <span>Tags</span>
        <input
          value={tagInput}
          onChange={(event) => setTagInput(event.target.value)}
          placeholder="pgvector, search, ollama"
        />
      </label>

      <button type="submit" className="primary-button" disabled={submitting}>
        {submitting ? "Saving..." : submitLabel}
      </button>
    </form>
  );
}
