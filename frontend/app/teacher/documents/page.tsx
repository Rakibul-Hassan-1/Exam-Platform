"use client";

import { useEffect, useRef, useState } from "react";
import { api, Subject, ApiError, GenerationJob } from "@/lib/api";

export default function DocumentsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [subjectId, setSubjectId] = useState("");
  const [sourceText, setSourceText] = useState("");
  const [count, setCount] = useState(10);
  const [easy, setEasy] = useState(30);
  const [medium, setMedium] = useState(50);
  const [hard, setHard] = useState(20);

  const [file, setFile] = useState<File | null>(null);

  const [status, setStatus] = useState<"idle" | "queued" | "polling" | "done" | "error">("idle");
  const [error, setError] = useState("");
  const [job, setJob] = useState<GenerationJob | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    api.listSubjects().then((r) => {
      setSubjects(r.subjects || []);
      if (r.subjects?.length) setSubjectId(r.subjects[0].id);
    });
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const pollJob = (jobId: string) => {
    setStatus("polling");
    pollRef.current = setInterval(async () => {
      try {
        const j = await api.jobStatus(jobId);
        setJob(j);
        if (j.status === "COMPLETED") {
          clearInterval(pollRef.current!);
          setStatus("done");
        } else if (j.status === "FAILED") {
          clearInterval(pollRef.current!);
          setStatus("error");
          setError(j.error || "Generation failed.");
        }
      } catch (e) {
        clearInterval(pollRef.current!);
        setStatus("error");
        setError(e instanceof ApiError ? e.message : "Failed to check job status.");
      }
    }, 2500);
  };

  const generateFromText = async () => {
    setError("");
    if (!subjectId) {
      setError("Choose a subject first.");
      return;
    }
    if (sourceText.trim().length < 200) {
      setError("Paste at least a few paragraphs (~200+ characters) so the AI has enough context.");
      return;
    }
    try {
      const { job } = await api.generateFromText({
        subject_id: subjectId,
        source_text: sourceText,
        question_count: count,
        difficulty: { easy, medium, hard },
      });
      setJob(job);
      setStatus("queued");
      pollJob(job.id);
    } catch (e) {
      setStatus("error");
      setError(e instanceof ApiError ? e.message : "Failed to queue generation job.");
    }
  };

  const uploadAndGenerate = async () => {
    setError("");
    if (!file || !subjectId) {
      setError("Choose a subject and a file first.");
      return;
    }
    try {
      const uploadRes = await api.uploadDocument(file, subjectId);
      if (uploadRes.warning) {
        setError(uploadRes.warning);
        return;
      }
      const { job } = await api.generateFromDocument(uploadRes.document.id, {
        subject_id: subjectId,
        question_count: count,
        difficulty: { easy, medium, hard },
      });
      setJob(job);
      setStatus("queued");
      pollJob(job.id);
    } catch (e) {
      setStatus("error");
      setError(e instanceof ApiError ? e.message : "Failed to upload or queue generation.");
    }
  };

  const reset = () => {
    setStatus("idle");
    setJob(null);
    setError("");
    setSourceText("");
    setFile(null);
  };

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">Generate questions with AI</h1>
      <p className="text-sm text-inksoft mb-4">
        Paste study material or upload a .txt/.md file. Generation runs asynchronously on the background
        worker — this page polls the job until it's done, then the questions land in the Question Bank as
        <span className="tag ml-1">REVIEWING</span>.
      </p>

      {subjects.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          Add a subject first, then come back here.
        </div>
      ) : status === "idle" || status === "error" ? (
        <div className="card space-y-4">
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Subject
            </label>
            <select className="input" value={subjectId} onChange={(e) => setSubjectId(e.target.value)}>
              {subjects.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Book / study material (paste text)
            </label>
            <textarea
              className="input font-mono text-xs"
              style={{ minHeight: 160 }}
              value={sourceText}
              onChange={(e) => setSourceText(e.target.value)}
              placeholder="Paste a chapter, article, or notes here…"
            />
            <div className="text-xs text-inksoft mt-1">{sourceText.length.toLocaleString()} characters</div>
          </div>

          <div className="text-center text-xs text-inksoft uppercase tracking-wide">— or —</div>

          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Upload a .txt or .md file instead
            </label>
            <input
              type="file"
              accept=".txt,.md,.pdf"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="text-sm"
            />
            <div className="text-xs text-inksoft mt-1">
              PDF upload is accepted but text extraction is a documented TODO in this scaffold — use pasted
              text for PDFs for now.
            </div>
          </div>

          <div className="flex gap-3 flex-wrap">
            <Field label="Number of questions">
              <input
                type="number"
                min={1}
                max={40}
                className="input w-24"
                value={count}
                onChange={(e) => setCount(Number(e.target.value) || 1)}
              />
            </Field>
            <Field label="Easy %">
              <input
                type="number"
                className="input w-20"
                value={easy}
                onChange={(e) => setEasy(Number(e.target.value) || 0)}
              />
            </Field>
            <Field label="Medium %">
              <input
                type="number"
                className="input w-20"
                value={medium}
                onChange={(e) => setMedium(Number(e.target.value) || 0)}
              />
            </Field>
            <Field label="Hard %">
              <input
                type="number"
                className="input w-20"
                value={hard}
                onChange={(e) => setHard(Number(e.target.value) || 0)}
              />
            </Field>
          </div>

          {error && <div className="text-pen text-sm">{error}</div>}

          <div className="flex gap-2">
            <button className="btn btn-brass" onClick={file ? uploadAndGenerate : generateFromText}>
              ✨ Generate
            </button>
          </div>
        </div>
      ) : (
        <div className="card text-center py-10">
          {status === "done" ? (
            <>
              <div className="text-good font-bold text-lg mb-1">Generation complete</div>
              <div className="text-sm text-inksoft mb-4">
                Questions were added to the Question Bank as REVIEWING — head over to review and approve them.
              </div>
              <a href="/teacher/questions" className="btn btn-primary">
                Go to Question Bank
              </a>
              <div className="mt-3">
                <button className="btn btn-ghost text-xs" onClick={reset}>
                  Generate another batch
                </button>
              </div>
            </>
          ) : (
            <>
              <div className="animate-pulse text-sm text-inksoft mb-2">
                {status === "queued" ? "Job queued…" : "Worker is processing your job…"}
              </div>
              <div className="text-xs text-inksoft">Job status: {job?.status || "PENDING"}</div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-[11px] uppercase tracking-wide text-inksoft font-semibold mb-1">{label}</div>
      {children}
    </label>
  );
}
