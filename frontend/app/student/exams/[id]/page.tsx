"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, Exam, QuestionPublic, ApiError } from "@/lib/api";

export default function TakeExamPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  const [exam, setExam] = useState<Exam | null>(null);
  const [questions, setQuestions] = useState<QuestionPublic[]>([]);
  const [deadline, setDeadline] = useState<Date | null>(null);
  const [idx, setIdx] = useState(0);
  const [answers, setAnswers] = useState<Record<string, number>>({});
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const submittedRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const detail = await api.getExam(id);
        if (cancelled) return;
        setExam(detail.exam);
        setQuestions(detail.questions as QuestionPublic[]);

        const start = await api.startExam(id);
        if (cancelled) return;
        setDeadline(new Date(start.deadline));
      } catch (e) {
        if (e instanceof ApiError && e.status === 409) {
          router.replace("/student/results");
          return;
        }
        setError(e instanceof ApiError ? e.message : "Failed to load exam.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const doSubmit = useCallback(async () => {
    if (submittedRef.current) return;
    submittedRef.current = true;
    try {
      const payload = questions.map((q) => ({
        question_id: q.id,
        selected_index: answers[q.id] ?? null,
      }));
      await api.submitExam(id, payload);
    } catch {
      // even on network hiccup, don't block the student from leaving the page
    } finally {
      router.replace("/student/results");
    }
  }, [answers, questions, id, router]);

  useEffect(() => {
    if (!deadline) return;
    const tick = () => {
      const secs = Math.max(0, Math.floor((deadline.getTime() - Date.now()) / 1000));
      setSecondsLeft(secs);
      if (secs <= 0) doSubmit();
    };
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [deadline, doSubmit]);

  if (loading) return <div className="text-center text-sm text-inksoft py-10">Loading exam…</div>;
  if (error) return <div className="text-center text-pen text-sm py-10">{error}</div>;
  if (!exam || questions.length === 0) return <div className="text-center text-sm text-inksoft py-10">This exam has no questions.</div>;

  const q = questions[idx];
  const mm = String(Math.floor(secondsLeft / 60)).padStart(2, "0");
  const ss = String(secondsLeft % 60).padStart(2, "0");
  const low = secondsLeft <= 60;

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <div>
          <div className="font-bold">{exam.title}</div>
          <div className="text-xs text-inksoft">
            Question {idx + 1} of {questions.length}
          </div>
        </div>
        <div
          className={`flex items-center gap-1.5 font-mono text-lg font-bold px-2.5 py-1 border rounded ${
            low ? "text-pen border-pen" : "border-line"
          }`}
        >
          ⏱ {mm}:{ss}
        </div>
      </div>

      <div className="flex gap-1.5 flex-wrap mb-4">
        {questions.map((qq, i) => (
          <button
            key={qq.id}
            onClick={() => setIdx(i)}
            className={`bubble ${i === idx ? "bubble-filled" : answers[qq.id] !== undefined ? "border-brass" : ""}`}
          >
            {i + 1}
          </button>
        ))}
      </div>

      <div className="card">
        <span className="tag">{q.difficulty}</span>
        <div className="font-bold text-base my-3">{q.question}</div>
        <div className="space-y-2">
          {q.options.map((o, i) => (
            <button
              key={i}
              onClick={() => setAnswers({ ...answers, [q.id]: i })}
              className={`w-full flex gap-2.5 items-center text-left px-3 py-2.5 rounded border ${
                answers[q.id] === i ? "border-ink bg-[#F1EFE4]" : "border-line"
              }`}
            >
              <span className={`bubble ${answers[q.id] === i ? "bubble-filled" : ""}`}>
                {String.fromCharCode(65 + i)}
              </span>
              {o}
            </button>
          ))}
        </div>
      </div>

      <div className="flex justify-between mt-4">
        <button className="btn btn-ghost" disabled={idx === 0} onClick={() => setIdx(Math.max(0, idx - 1))}>
          ← Previous
        </button>
        {idx < questions.length - 1 ? (
          <button className="btn btn-primary" onClick={() => setIdx(idx + 1)}>
            Next →
          </button>
        ) : (
          <button className="btn btn-brass" onClick={doSubmit}>
            ✓ Submit exam
          </button>
        )}
      </div>
    </div>
  );
}
