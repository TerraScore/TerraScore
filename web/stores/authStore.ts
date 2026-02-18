import { create } from "zustand";
import type { UserProfile } from "@/lib/types";

interface AuthState {
  isAuthenticated: boolean;
  profile: UserProfile | null;
  isLoading: boolean;
  setAuthenticated: (profile: UserProfile) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  profile: null,
  isLoading: true,
  setAuthenticated: (profile) =>
    set({ isAuthenticated: true, profile, isLoading: false }),
  clearAuth: () =>
    set({ isAuthenticated: false, profile: null, isLoading: false }),
  setLoading: (loading) => set({ isLoading: loading }),
}));
