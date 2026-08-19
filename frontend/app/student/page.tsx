"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, Exam, Subject, ExamAttempt } from "@/lib/api";

export default function StudentExamsPage() {
  const [exams, setExams] = useState<Exam[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [results, setResults] = useState<ExamAttempt[]>([]);
  const router = useRouter();

  useEffect(() => {
    api.listExams().then((r) => setExams(r.exams || []));
    api.listSubjects().then((r) => setSubjects(r.subjects || []));
    api.listResults().then((r) => setResults(r.results || []));
  }, []);

  const subjectName = (id: string) => subjects.find((s) => s.id === id)?.name || "—";
  const taken = (examId: string) => results.some((r) => r.exam_id === examId);

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">Available exams</h1>
      <p className="text-sm text-inksoft mb-4">Published examinations you can take now.</p>

      {exams.length === 0 ? (
        <div className="text-center py-8 px-3 text-inksoft text-sm border border-dashed border-line rounded">
          No exams have been published yet. Check back soon.
        </div>
      ) : (
        <div className="space-y-3">
          {exams.map((e) => (
            <div key={e.id} className="card">
              <div className="flex justify-between items-start">
                <div>
                  <div className="font-bold">{e.title}</div>
                  <div className="text-xs text-inksoft mt-1">
                    {subjectName(e.subject_id)} · {e.total_marks} questions · {e.duration_min} min ·{" "}
                    {e.total_marks} marks
                  </div>
                </div>
                {taken(e.id) && <span className="tag text-good bg-[#E4F0E9]">Completed</span>}
              </div>
              <button
                className="btn btn-primary text-xs mt-3"
                disabled={taken(e.id)}
                onClick={() => router.push(`/student/exams/${e.id}`)}
              >
                {taken(e.id) ? "Already submitted" : "Start exam →"}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
