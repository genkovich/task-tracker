import { createContext, useCallback, useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { BASE_URL, setOnUnauthorized } from "@/shared/api/client";
import { userApi } from "@/entities/user/api/userApi";
import type { CurrentUser } from "@/entities/user/model/types";

interface AuthContextValue {
  user: CurrentUser | null;
  isAdmin: boolean;
  isLoading: boolean;
  login: () => void;
  logout: () => void;
  setTokens: (accessToken: string, refreshToken: string) => void;
  fetchUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const isServer = typeof window === "undefined";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const navigate = useNavigate();

  const fetchUser = useCallback(async () => {
    try {
      const currentUser = await userApi.getCurrentUser();
      setUser(currentUser);
    } catch {
      setUser(null);
    }
  }, []);

  const login = useCallback(() => {
    if (!isServer) {
      window.location.href = `${BASE_URL}/api/v1/auth/google`;
    }
  }, []);

  const logout = useCallback(() => {
    if (!isServer) {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
    }
    setUser(null);
    navigate("/login");
  }, [navigate]);

  useEffect(() => {
    if (isServer) {
      setIsLoading(false);
      return;
    }
    const token = localStorage.getItem("access_token");
    if (token) {
      fetchUser().finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [fetchUser]);

  useEffect(() => {
    setOnUnauthorized(logout);
  }, [logout]);

  const setTokens = useCallback((accessToken: string, refreshToken: string) => {
    if (!isServer) {
      localStorage.setItem("access_token", accessToken);
      localStorage.setItem("refresh_token", refreshToken);
    }
  }, []);

  const isAdmin = user?.role === "admin";

  return (
    <AuthContext
      value={{ user, isAdmin: !!isAdmin, isLoading, login, logout, setTokens, fetchUser }}
    >
      {children}
    </AuthContext>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
