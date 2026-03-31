"use client";

import { createContext, useContext, useEffect, useState, useCallback, ReactNode } from "react";
import { getMe, login as apiLogin, logout as apiLogout } from "@/lib/api";

interface User {
  id: string;
  name: string;
  email: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
  updateUser: (updates: Partial<User>) => void;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  login: async () => {},
  logout: async () => {},
  refreshUser: async () => {},
  updateUser: () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshUser = useCallback(async () => {
    try {
      const u = await getMe();
      setUser(u);
    } catch {
      setUser(null);
    }
  }, []);

  useEffect(() => {
    refreshUser().finally(() => setLoading(false));
  }, [refreshUser]);

  const login = async (email: string, password: string) => {
    const res = await apiLogin(email, password);
    setUser(res.user);
    // HTTP-only cookie is set automatically by the server response
  };

  const logout = async () => {
    try {
      await apiLogout();
    } catch {}
    setUser(null);
  };

  const updateUser = (updates: Partial<User>) => {
    setUser((prev: User | null) => (prev ? { ...prev, ...updates } : null));
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshUser, updateUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
