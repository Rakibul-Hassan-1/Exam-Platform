"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { ApiError } from "@/lib/api";

export default function RegisterPage() {
  const { register } = useAuth();
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"teacher" | "student">("teacher");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await register(name, email, password, role);
      router.push("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Registration failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen bg-ink flex items-center justify-center px-5">
      <div className="w-full max-w-sm">
        <div className="text-center mb-6">
          <div className="text-[#F6F5EF] text-2xl">Examination Hall</div>
          <div className="text-[#9FB3A7] text-xs uppercase tracking-widest mt-1">Create an account</div>
        </div>
        <form onSubmit={submit} className="bg-[#F6F5EF] rounded p-6 space-y-4">
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">Name</label>
            <input required className="input" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">Email</label>
            <input type="email" required className="input" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Password (min. 8 characters)
            </label>
            <input
              type="password"
              required
              minLength={8}
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              I am a
            </label>
            <div className="flex gap-2">
              {(["teacher", "student"] as const).map((r) => (
                <button
                  type="button"
                  key={r}
                  onClick={() => setRole(r)}
                  className={`flex-1 py-2 rounded border text-sm font-semibold capitalize ${
                    role === r ? "bg-ink text-[#F6F5EF] border-ink" : "bg-white border-line text-ink"
                  }`}
                >
                  {r}
                </button>
              ))}
            </div>
          </div>
          {error && <div className="text-pen text-sm">{error}</div>}
          <button className="btn btn-primary w-full justify-center" disabled={busy} type="submit">
            {busy ? "Creating account…" : "Create account"}
          </button>
          <div className="text-center text-sm text-inksoft">
            Already have an account?{" "}
            <Link href="/login" className="text-brassdk font-semibold underline">
              Sign in
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
