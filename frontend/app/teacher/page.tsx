"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, Subject, Question, Exam, ExamAttempt } from "@/lib/api";

export default function TeacherOverview() {
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [exams, setExams] = useState<Exam[]>([]);
  const [results, setResults] = useState<ExamAttempt[]>([]);

  useEffect(() => {
    api.listSubjects().then((r) => setSubjects(r.subjects || []));
    api.listQuestions().then((r) => setQuestions(r.questions || []));
    api.listExams().then((r) => setExams(r.exams || []));
    api.listResults().then((r) => setResults(r.results || []));
  }, []);

  const approved = questions.filter((q) => q.status === "APPROVED").length;
  const aiGenerated = questions.filter((q) => q.source === "ai").length;
  const published = exams.filter((e) => e.published).length;

  const stats = [
    { label: "Subjects", value: subjects.length },
    { label: "Approved Qs", value: approved },
    { label: "AI generated", value: aiGenerated },
    { label: "Published exams", value: published },
    { label: "Submissions", value: results.length },
  ];

  return (
    <div>
      <h1 className="text-xl font-bold mb-1">Console overview</h1>
      <p className="text-sm text-inksoft mb-4">
        Everything happening across the shared question bank and exam schedule.
      </p>

      <div className="flex gap-3 flex-wrap mb-5">
        {stats.map((s) => (
          <div key={s.label} className="card flex-1 min-w-[110px] text-center">
            <div className="font-mono text-2xl font-bold">{s.value}</div>
            <div className="text-[11px] text-inksoft uppercase tracking-wide mt-1">{s.label}</div>
          </div>
        ))}
      </div>

      <div className="card">
        <div className="font-bold text-sm mb-2">Quick start</div>
        <ol className="list-decimal pl-5 text-sm text-inksoft space-y-2">
          <li>
            Add a <Link href="/teacher/subjects" className="text-brassdk underline">subject</Link>.
          </li>
          <li>
            Paste study material into <Link href="/teacher/documents" className="text-brassdk underline">Generate (AI)</Link> to
            create questions, or add them manually in the Question Bank.
          </li>
          <li>Review, edit, and approve generated questions.</li>
          <li>
            Build an <Link href="/teacher/exams" className="text-brassdk underline">exam</Link> from approved questions and
            publish it.
          </li>
          <li>Students take the exam; results appear automatically in Results.</li>
        </ol>
      </div>
    </div>
  );
}
