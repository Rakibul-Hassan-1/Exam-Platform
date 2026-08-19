"use client";

import { useEffect, useState } from "react";
import { api, Subject, Question, Exam, ApiError } from "@/lib/api";

export default function ExamsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [exams, setExams] = useState<Exam[]>([]);
  const [creating, setCreating] = useState(false);

  const load = async () => {
    const [s, q, e] = await Promise.all([
      api.listSubjects(),
      api.listQuestions({ status: "APPROVED" }),
      api.listExams(),
    ]);
    setSubjects(s.subjects || []);
    setQuestions(q.questions || []);
    setExams(e.exams || []);
  };
  useEffect(() => {
    load();
  }, []);

  const subjectName = (id: string) => subjects.find((s) => s.id === id)?.name || "—";

  const togglePublish = async (exam: Exam) => {
    await api.setPublished(exam.id, !exam.published);
    load();
  };
  const remove = async (id: string) => {
    await api.deleteExam(id);
    load();
  };

  return (
    <div>
      <div className="flex justify-between items-end mb-4">
        <div>
          <h1 className="text-xl font-bold">Exams</h1>
          <p className="text-sm text-inksoft">Assemble approved questions into a timed examination.</p>
        </div>
        <button className="btn btn-brass text-xs" onClick={() => setCreating(!creating)}>
          + New exam
        </button>
      </div>

      {creating && (
        <ExamBuilder
          subjects={subjects}
          questions={questions}
          onCreate={async (input) => {
            await api.createExam(input);
            setCreating(false);
            load();
          }}
          onCancel={() => setCreating(false)}
        />
      )}

      {exams.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          No exams yet.
        </div>
      ) : (
        <div className="space-y-3">
          {exams.map((e) => (
            <div key={e.id} className="card">
              <div className="flex justify-between items-start gap-3">
                <div>
                  <div className="font-bold">{e.title}</div>
                  <div className="text-xs text-inksoft mt-1">
                    {subjectName(e.subject_id)} · {e.total_marks} questions · {e.duration_min} min ·{" "}
                    {e.total_marks} marks
                  </div>
                </div>
                <span className={`tag ${e.published ? "text-good bg-[#E4F0E9]" : ""}`}>
                  {e.published ? "Published" : "Draft"}
                </span>
              </div>
              <div className="flex gap-2 mt-3">
                <button className="btn btn-primary text-xs" onClick={() => togglePublish(e)}>
                  {e.published ? "Unpublish" : "Publish"}
                </button>
                <button className="btn btn-danger text-xs" onClick={() => remove(e.id)}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ExamBuilder({
  subjects,
  questions,
  onCreate,
  onCancel,
}: {
  subjects: Subject[];
  questions: Question[];
  onCreate: (input: { title: string; subject_id: string; duration_min: number; question_ids: string[] }) => void;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState("");
  const [subjectId, setSubjectId] = useState(subjects[0]?.id || "");
  const [duration, setDuration] = useState(30);
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState("");

  const pool = questions.filter((q) => q.subject_id === subjectId);
  const toggle = (id: string) =>
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const create = () => {
    if (!title.trim() || selected.length === 0) {
      setError("Give the exam a title and select at least one question.");
      return;
    }
    onCreate({ title: title.trim(), subject_id: subjectId, duration_min: duration, question_ids: selected });
  };

  return (
    <div className="card mb-4 border-brass">
      <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
        Exam title
      </label>
      <input className="input mb-3" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Data Structures Midterm Exam" />

      <div className="flex gap-3 mb-3">
        <div className="flex-1">
          <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
            Subject
          </label>
          <select
            className="input"
            value={subjectId}
            onChange={(e) => {
              setSubjectId(e.target.value);
              setSelected([]);
            }}
          >
            {subjects.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
            Duration (min)
          </label>
          <input
            type="number"
            className="input w-24"
            value={duration}
            onChange={(e) => setDuration(Number(e.target.value) || 5)}
          />
        </div>
      </div>

      <div className="text-xs uppercase tracking-wide text-inksoft font-semibold mb-2">
        Choose approved questions ({selected.length} selected)
      </div>
      {pool.length === 0 ? (
        <div className="text-center py-6 text-inksoft text-sm border border-dashed border-line rounded mb-3">
          No approved questions for this subject yet.
        </div>
      ) : (
        <div className="space-y-1.5 max-h-64 overflow-y-auto mb-3">
          {pool.map((q) => (
            <label
              key={q.id}
              className={`flex gap-2 items-start text-sm p-2 border rounded cursor-pointer ${
                selected.includes(q.id) ? "border-brass bg-[#FBF6EA]" : "border-line"
              }`}
            >
              <input type="checkbox" checked={selected.includes(q.id)} onChange={() => toggle(q.id)} className="mt-1" />
              <span>
                {q.question} <span className="tag ml-1">{q.difficulty}</span>
              </span>
            </label>
          ))}
        </div>
      )}
      {error && <div className="text-pen text-sm mb-2">{error}</div>}
      <div className="flex gap-2">
        <button className="btn btn-primary" onClick={create}>
          Create exam
        </button>
        <button className="btn btn-ghost" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}
