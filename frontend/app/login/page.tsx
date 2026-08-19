"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { ApiError } from "@/lib/api";

export default function LoginPage() {
  const { login } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await login(email, password);
      router.push("/");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen bg-ink flex items-center justify-center px-5">
      <div className="w-full max-w-sm">
        <div className="text-center mb-6">
          <div className="text-[#F6F5EF] text-2xl">Examination Hall</div>
          <div className="text-[#9FB3A7] text-xs uppercase tracking-widest mt-1">
            AI Question Generation Platform
          </div>
        </div>
        <form onSubmit={submit} className="bg-[#F6F5EF] rounded p-6 space-y-4">
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Email
            </label>
            <input
              type="email"
              required
              className="input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs uppercase tracking-wide text-inksoft font-semibold mb-1">
              Password
            </label>
            <input
              type="password"
              required
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          {error && <div className="text-pen text-sm">{error}</div>}
          <button className="btn btn-primary w-full justify-center" disabled={busy} type="submit">
            {busy ? "Signing in…" : "Sign in"}
          </button>
          <div className="text-center text-sm text-inksoft">
            No account?{" "}
            <Link href="/register" className="text-brassdk font-semibold underline">
              Register
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
