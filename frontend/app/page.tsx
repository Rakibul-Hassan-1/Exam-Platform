"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";

export default function Home() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/login");
    } else if (user.role === "teacher" || user.role === "admin") {
      router.replace("/teacher");
    } else {
      router.replace("/student");
    }
  }, [user, loading, router]);

  return (
    <div className="min-h-screen flex items-center justify-center text-inksoft text-sm">
      Loading Examination Hall…
    </div>
  );
}
