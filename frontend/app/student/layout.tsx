"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";

const TABS = [
  { href: "/student", label: "Exams" },
  { href: "/student/results", label: "My Results" },
];

export default function StudentLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (loading) return;
    if (!user) router.replace("/login");
    else if (user.role !== "student") router.replace("/teacher");
  }, [user, loading, router]);

  if (loading || !user) {
    return <div className="min-h-screen flex items-center justify-center text-inksoft text-sm">Loading…</div>;
  }

  return (
    <div className="min-h-screen bg-paper">
      <div className="flex items-center justify-between px-5 py-3 border-b border-line bg-white sticky top-0 z-10">
        <div>
          <div className="font-bold text-[15px] leading-tight">Examination Hall</div>
          <div className="text-[10.5px] text-inksoft uppercase tracking-wide">Student portal</div>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-inksoft">{user.name}</span>
          <button onClick={logout} className="btn btn-ghost text-xs">
            Exit
          </button>
        </div>
      </div>
      <div className="flex gap-1 px-5 pt-3 overflow-x-auto border-b border-line bg-white">
        {TABS.map((t) => (
          <Link
            key={t.href}
            href={t.href}
            className={`px-3 py-2 text-sm font-semibold whitespace-nowrap rounded-t border ${
              pathname === t.href ? "bg-paper border-line border-b-paper text-ink" : "border-transparent text-inksoft"
            }`}
          >
            {t.label}
          </Link>
        ))}
      </div>
      <div className="max-w-xl mx-auto p-5">{children}</div>
    </div>
  );
}
