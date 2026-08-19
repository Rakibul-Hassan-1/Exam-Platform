"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, setToken, User } from "./api";

interface AuthState {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string, role: "teacher" | "student") => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const stored = typeof window !== "undefined" ? window.localStorage.getItem("exam_platform_user") : null;
    if (stored) {
      try {
        setUser(JSON.parse(stored));
      } catch {
        /* ignore */
      }
    }
    setLoading(false);
  }, []);

  const persist = (u: User, token: string) => {
    setToken(token);
    window.localStorage.setItem("exam_platform_user", JSON.stringify(u));
    setUser(u);
  };

  const login = async (email: string, password: string) => {
    const res = await api.login({ email, password });
    persist(res.user, res.token);
  };

  const register = async (name: string, email: string, password: string, role: "teacher" | "student") => {
    const res = await api.register({ name, email, password, role });
    persist(res.user, res.token);
  };

  const logout = () => {
    setToken(null);
    window.localStorage.removeItem("exam_platform_user");
    setUser(null);
    router.push("/login");
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
