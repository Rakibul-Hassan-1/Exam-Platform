"use client";

import { useEffect, useState } from "react";
import { api, ExamAttempt, Exam } from "@/lib/api";

export default function TeacherResultsPage() {
  const [results, setResults] = useState<ExamAttempt[]>([]);
  const [exams, setExams] = useState<Exam[]>([]);

  useEffect(() => {
    api.listResults().then((r) => setResults(r.results || []));
    api.listExams().then((r) => setExams(r.exams || []));
  }, []);

  const examTitle = (id: string) => exams.find((e) => e.id === id)?.title || "Deleted exam";

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">Results</h1>
      <p className="text-sm text-inksoft mb-4">Every student submission across all exams.</p>

      {results.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          No submissions yet.
        </div>
      ) : (
        <div className="space-y-2">
          {results.map((a) => {
            const pct = Math.round(a.percentage);
            return (
              <div key={a.id} className="card flex justify-between items-center">
                <div>
                  <div className="font-bold text-sm">{examTitle(a.exam_id)}</div>
                  <div className="text-xs text-inksoft">
                    Student: {a.student_id.slice(0, 8)}… ·{" "}
                    {a.submitted_at ? new Date(a.submitted_at).toLocaleString() : "—"} ·{" "}
                    <span className="tag">{a.status}</span>
                  </div>
                </div>
                <div className={`font-mono text-lg font-bold ${pct >= 50 ? "text-good" : "text-pen"}`}>
                  {a.correct_count}/{a.total_count} · {pct}%
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
