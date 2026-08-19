"use client";

import { useEffect, useState } from "react";
import { api, Subject, ApiError } from "@/lib/api";

export default function SubjectsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = () => api.listSubjects().then((r) => setSubjects(r.subjects || []));
  useEffect(() => {
    load();
  }, []);

  const add = async () => {
    if (!name.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api.createSubject(name.trim());
      setName("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to add subject");
    } finally {
      setBusy(false);
    }
  };

  const remove = async (id: string) => {
    await api.deleteSubject(id);
    await load();
  };

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">Subjects</h1>
      <p className="text-sm text-inksoft mb-4">Every exam and question belongs to a subject.</p>

      <div className="card mb-4">
        <div className="flex gap-2">
          <input
            className="input"
            placeholder="e.g. Data Structures"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && add()}
          />
          <button className="btn btn-primary" disabled={busy} onClick={add}>
            + Add
          </button>
        </div>
        {error && <div className="text-pen text-sm mt-2">{error}</div>}
      </div>

      {subjects.length === 0 ? (
        <Empty text="No subjects yet. Add your first one above." />
      ) : (
        <div className="space-y-2">
          {subjects.map((s) => (
            <div key={s.id} className="card flex justify-between items-center py-2.5 px-3.5">
              <span className="font-semibold">{s.name}</span>
              <button className="btn btn-danger text-xs" onClick={() => remove(s.id)}>
                Delete
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function Empty({ text }: { text: string }) {
  return (
    <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
      {text}
    </div>
  );
}
