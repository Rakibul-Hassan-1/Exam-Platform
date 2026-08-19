// Thin fetch wrapper around the Go backend's REST API. Attaches the
// JWT bearer token from localStorage and normalizes error handling.

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type Role = "teacher" | "student" | "admin";

export interface User {
  id: string;
  name: string;
  email: string;
  role: Role;
}

export interface Subject {
  id: string;
  name: string;
  created_by: string;
  created_at: string;
}

export interface Question {
  id: string;
  subject_id: string;
  question: string;
  options: string[];
  correct_index: number;
  difficulty: "easy" | "medium" | "hard";
  explanation: string;
  source: "manual" | "ai";
  status: "GENERATED" | "REVIEWING" | "APPROVED" | "REJECTED" | "ARCHIVED";
  created_by: string;
  created_at: string;
}

export interface QuestionPublic {
  id: string;
  question: string;
  options: string[];
  difficulty: "easy" | "medium" | "hard";
}

export interface Exam {
  id: string;
  title: string;
  subject_id: string;
  duration_min: number;
  total_marks: number;
  passing_marks: number;
  published: boolean;
  created_by: string;
  created_at: string;
  question_ids?: string[];
}

export interface ExamAttempt {
  id: string;
  exam_id: string;
  student_id: string;
  status: "IN_PROGRESS" | "SUBMITTED" | "AUTO_SUBMITTED";
  started_at: string;
  submitted_at?: string;
  correct_count: number;
  total_count: number;
  obtained_marks: number;
  total_marks: number;
  percentage: number;
}

export interface GenerationJob {
  id: string;
  status: "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";
  question_count: number;
  error?: string;
}

class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem("exam_platform_token");
}

export function setToken(token: string | null) {
  if (typeof window === "undefined") return;
  if (token) window.localStorage.setItem("exam_platform_token", token);
  else window.localStorage.removeItem("exam_platform_token");
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    ...(options.body && !(options.body instanceof FormData)
      ? { "Content-Type": "application/json" }
      : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...((options.headers as Record<string, string>) || {}),
  };

  const res = await fetch(`${API_URL}${path}`, { ...options, headers });

  if (res.status === 204) return undefined as unknown as T;

  let data: any = null;
  try {
    data = await res.json();
  } catch {
    // no body
  }

  if (!res.ok) {
    throw new ApiError(data?.error || `Request failed (${res.status})`, res.status);
  }
  return data as T;
}

export const api = {
  // --- auth ---
  register: (body: { name: string; email: string; password: string; role: Role }) =>
    request<{ token: string; user: User }>("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  login: (body: { email: string; password: string }) =>
    request<{ token: string; user: User }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => request<{ id: string; name: string; role: Role }>("/api/v1/auth/me"),

  // --- subjects ---
  listSubjects: () => request<{ subjects: Subject[] }>("/api/v1/subjects"),
  createSubject: (name: string) =>
    request<Subject>("/api/v1/subjects", { method: "POST", body: JSON.stringify({ name }) }),
  deleteSubject: (id: string) => request<void>(`/api/v1/subjects/${id}`, { method: "DELETE" }),

  // --- questions ---
  listQuestions: (params: { subject_id?: string; status?: string } = {}) => {
    const qs = new URLSearchParams(params as Record<string, string>).toString();
    return request<{ questions: Question[] }>(`/api/v1/questions${qs ? `?${qs}` : ""}`);
  },
  createQuestion: (body: {
    subject_id: string;
    question: string;
    options: string[];
    correct_index: number;
    difficulty: string;
  }) => request<Question>("/api/v1/questions", { method: "POST", body: JSON.stringify(body) }),
  updateQuestion: (id: string, body: Partial<Question>) =>
    request<Question>(`/api/v1/questions/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteQuestion: (id: string) => request<void>(`/api/v1/questions/${id}`, { method: "DELETE" }),

  // --- documents / AI generation ---
  uploadDocument: (file: File, subjectId: string) => {
    const form = new FormData();
    form.append("file", file);
    form.append("subject_id", subjectId);
    return request<{ document: any; chunks?: number; warning?: string }>("/api/v1/documents/upload", {
      method: "POST",
      body: form,
    });
  },
  generateFromDocument: (
    documentId: string,
    body: { subject_id: string; question_count: number; difficulty: { easy: number; medium: number; hard: number } }
  ) =>
    request<{ job: GenerationJob }>(`/api/v1/documents/${documentId}/generate-questions`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  generateFromText: (body: {
    subject_id: string;
    source_text: string;
    question_count: number;
    difficulty: { easy: number; medium: number; hard: number };
  }) =>
    request<{ job: GenerationJob }>("/api/v1/documents/generate-from-text", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  jobStatus: (id: string) => request<GenerationJob>(`/api/v1/generation-jobs/${id}`),

  // --- exams ---
  listExams: () => request<{ exams: Exam[] }>("/api/v1/exams"),
  createExam: (body: { title: string; subject_id: string; duration_min: number; question_ids: string[] }) =>
    request<Exam>("/api/v1/exams", { method: "POST", body: JSON.stringify(body) }),
  getExam: (id: string) =>
    request<{ exam: Exam; questions: Question[] | QuestionPublic[] }>(`/api/v1/exams/${id}`),
  setPublished: (id: string, published: boolean) =>
    request<{ id: string; published: boolean }>(`/api/v1/exams/${id}/publish`, {
      method: "PATCH",
      body: JSON.stringify({ published }),
    }),
  deleteExam: (id: string) => request<void>(`/api/v1/exams/${id}`, { method: "DELETE" }),

  // --- attempts ---
  startExam: (id: string) =>
    request<{ attempt: ExamAttempt; started_at: string; deadline: string }>(`/api/v1/exams/${id}/start`, {
      method: "POST",
    }),
  submitExam: (id: string, answers: { question_id: string; selected_index: number | null }[]) =>
    request<ExamAttempt>(`/api/v1/exams/${id}/submit`, {
      method: "POST",
      body: JSON.stringify({ answers }),
    }),

  // --- results ---
  listResults: () => request<{ results: ExamAttempt[] }>("/api/v1/results"),
};

export { ApiError };
