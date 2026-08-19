"use client";

import { useEffect, useState } from "react";
import { api, ExamAttempt, Exam } from "@/lib/api";

export default function StudentResultsPage() {
  const [results, setResults] = useState<ExamAttempt[]>([]);
  const [exams, setExams] = useState<Exam[]>([]);

  useEffect(() => {
    api.listResults().then((r) => setResults(r.results || []));
    api.listExams().then((r) => setExams(r.exams || []));
  }, []);

  const examTitle = (id: string) => exams.find((e) => e.id === id)?.title || "Deleted exam";

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">My results</h1>
      <p className="text-sm text-inksoft mb-4">Your examination history and scores.</p>

      {results.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          You haven't submitted any exams yet.
        </div>
      ) : (
        <div className="space-y-2">
          {results.map((a) => {
            const pct = Math.round(a.percentage);
            return (
              <div key={a.id} className="card flex justify-between items-center">
                <div>
                  <div className="font-bold">{examTitle(a.exam_id)}</div>
                  <div className="text-xs text-inksoft">
                    {a.submitted_at ? new Date(a.submitted_at).toLocaleString() : "—"} · {a.correct_count}/
                    {a.total_count} correct
                  </div>
                </div>
                <div className={`font-mono text-lg font-bold ${pct >= 50 ? "text-good" : "text-pen"}`}>{pct}%</div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
