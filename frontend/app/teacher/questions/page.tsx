"use client";

import { useEffect, useState } from "react";
import { api, Subject, Question, ApiError } from "@/lib/api";

export default function QuestionsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [filterSubject, setFilterSubject] = useState("all");
  const [filterStatus, setFilterStatus] = useState("all");
  const [showManualForm, setShowManualForm] = useState(false);

  const load = async () => {
    const [s, q] = await Promise.all([
      api.listSubjects(),
      api.listQuestions({
        subject_id: filterSubject !== "all" ? filterSubject : undefined,
        status: filterStatus !== "all" ? filterStatus : undefined,
      }),
    ]);
    setSubjects(s.subjects || []);
    setQuestions(q.questions || []);
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterSubject, filterStatus]);

  const subjectName = (id: string) => subjects.find((s) => s.id === id)?.name || "—";

  const setStatus = async (id: string, status: string) => {
    await api.updateQuestion(id, { status: status as any });
    load();
  };
  const remove = async (id: string) => {
    await api.deleteQuestion(id);
    load();
  };

  return (
    <div>
      <div className="flex justify-between items-end mb-4 gap-3">
        <div>
          <h1 className="text-xl font-bold">Question Bank</h1>
          <p className="text-sm text-inksoft">Manual and AI-generated questions, reviewed before use.</p>
        </div>
        <button className="btn btn-ghost text-xs" onClick={() => setShowManualForm(!showManualForm)}>
          + Manual question
        </button>
      </div>

      {showManualForm && (
        <ManualForm subjects={subjects} onDone={() => { setShowManualForm(false); load(); }} />
      )}

      <div className="flex gap-2 flex-wrap mb-3">
        <select className="input w-auto" value={filterSubject} onChange={(e) => setFilterSubject(e.target.value)}>
          <option value="all">All subjects</option>
          {subjects.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
        <select className="input w-auto" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}>
          <option value="all">All statuses</option>
          {["GENERATED", "REVIEWING", "APPROVED", "REJECTED"].map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      {questions.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          No questions match this filter yet.
        </div>
      ) : (
        <div className="space-y-3">
          {questions.map((q) => (
            <QuestionCard
              key={q.id}
              q={q}
              subjectName={subjectName(q.subject_id)}
              onApprove={() => setStatus(q.id, "APPROVED")}
              onReject={() => setStatus(q.id, "REJECTED")}
              onDelete={() => remove(q.id)}
              onSaved={load}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ManualForm({ subjects, onDone }: { subjects: Subject[]; onDone: () => void }) {
  const [subjectId, setSubjectId] = useState(subjects[0]?.id || "");
  const [text, setText] = useState("");
  const [options, setOptions] = useState(["", "", "", ""]);
  const [correct, setCorrect] = useState(0);
  const [difficulty, setDifficulty] = useState("medium");
  const [error, setError] = useState("");

  const save = async () => {
    if (!subjectId || !text.trim() || options.some((o) => !o.trim())) {
      setError("Fill in the question, all 4 options, and pick a subject.");
      return;
    }
    try {
      await api.createQuestion({
        subject_id: subjectId,
        question: text.trim(),
        options: options.map((o) => o.trim()),
        correct_index: correct,
        difficulty,
      });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to save question.");
    }
  };

  return (
    <div className="card mb-4 border-brass">
      <div className="font-bold mb-2">New manual question</div>
      <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">Subject</label>
      <select className="input mb-3" value={subjectId} onChange={(e) => setSubjectId(e.target.value)}>
        {subjects.map((s) => (
          <option key={s.id} value={s.id}>
            {s.name}
          </option>
        ))}
      </select>
      <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">Question</label>
      <textarea className="input mb-3" value={text} onChange={(e) => setText(e.target.value)} />
      <div className="text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
        Options (select the correct one)
      </div>
      {options.map((o, i) => (
        <div key={i} className="flex gap-2 mb-2 items-center">
          <button
            type="button"
            onClick={() => setCorrect(i)}
            className={`bubble ${correct === i ? "bubble-filled" : ""}`}
          >
            {String.fromCharCode(65 + i)}
          </button>
          <input
            className="input"
            value={o}
            onChange={(e) => {
              const next = [...options];
              next[i] = e.target.value;
              setOptions(next);
            }}
          />
        </div>
      ))}
      <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1 mt-2">
        Difficulty
      </label>
      <select className="input mb-3" value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
        <option value="easy">Easy</option>
        <option value="medium">Medium</option>
        <option value="hard">Hard</option>
      </select>
      {error && <div className="text-pen text-sm mb-2">{error}</div>}
      <div className="flex gap-2">
        <button className="btn btn-primary" onClick={save}>
          Save question
        </button>
      </div>
    </div>
  );
}

function QuestionCard({
  q,
  subjectName,
  onApprove,
  onReject,
  onDelete,
  onSaved,
}: {
  q: Question;
  subjectName: string;
  onApprove: () => void;
  onReject: () => void;
  onDelete: () => void;
  onSaved: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(q);

  const save = async () => {
    await api.updateQuestion(q.id, {
      question: draft.question,
      options: draft.options,
      correct_index: draft.correct_index,
    });
    setEditing(false);
    onSaved();
  };

  const statusColors: Record<string, string> = {
    GENERATED: "text-brassdk bg-[#F5EEDC]",
    REVIEWING: "text-brassdk bg-[#F5EEDC]",
    APPROVED: "text-good bg-[#E4F0E9]",
    REJECTED: "text-pen bg-[#F6E6E4]",
  };

  if (editing) {
    return (
      <div className="card border-brass">
        <textarea
          className="input mb-2"
          value={draft.question}
          onChange={(e) => setDraft({ ...draft, question: e.target.value })}
        />
        {draft.options.map((o, i) => (
          <div key={i} className="flex gap-2 mb-2 items-center">
            <button
              type="button"
              onClick={() => setDraft({ ...draft, correct_index: i })}
              className={`bubble ${draft.correct_index === i ? "bubble-filled" : ""}`}
            >
              {String.fromCharCode(65 + i)}
            </button>
            <input
              className="input"
              value={o}
              onChange={(e) => {
                const next = [...draft.options];
                next[i] = e.target.value;
                setDraft({ ...draft, options: next });
              }}
            />
          </div>
        ))}
        <div className="flex gap-2 mt-2">
          <button className="btn btn-primary text-xs" onClick={save}>
            Save
          </button>
          <button className="btn btn-ghost text-xs" onClick={() => { setDraft(q); setEditing(false); }}>
            Cancel
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="card">
      <div className="flex gap-1.5 flex-wrap mb-2">
        <span className="tag">{subjectName}</span>
        <span className="tag">{q.difficulty}</span>
        {q.source === "ai" && <span className="tag text-brassdk bg-[#F5EEDC]">✨ AI</span>}
        <span className={`tag ${statusColors[q.status] || ""}`}>{q.status}</span>
      </div>
      <div className="font-semibold mb-2">{q.question}</div>
      <div className="grid grid-cols-2 gap-1.5 mb-2">
        {q.options.map((o, i) => (
          <div
            key={i}
            className={`flex gap-1.5 items-center text-xs ${
              i === q.correct_index ? "text-good font-bold" : "text-inksoft"
            }`}
          >
            <span className={`bubble ${i === q.correct_index ? "bubble-correct" : ""}`} style={{ width: 22, height: 22, fontSize: 10 }}>
              {String.fromCharCode(65 + i)}
            </span>
            {o}
          </div>
        ))}
      </div>
      {q.explanation && <div className="text-xs text-inksoft italic mb-2">{q.explanation}</div>}
      <div className="flex gap-1.5 flex-wrap">
        {q.status !== "APPROVED" && (
          <button className="btn btn-primary text-xs" onClick={onApprove}>
            ✓ Approve
          </button>
        )}
        {q.status !== "REJECTED" && (
          <button className="btn btn-ghost text-xs" onClick={onReject}>
            ✕ Reject
          </button>
        )}
        <button className="btn btn-ghost text-xs" onClick={() => setEditing(true)}>
          Edit
        </button>
        <button className="btn btn-danger text-xs" onClick={onDelete}>
          Delete
        </button>
      </div>
    </div>
  );
}
