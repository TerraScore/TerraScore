import { create } from 'zustand';
import { getStoredToken, clearTokens, verifyOTP, requestOTP, registerAgent } from '@/services/auth';
import type { RegisterAgentParams } from '@/services/auth';
import { api } from '@/services/api';
import type { AgentProfile, ApiResponse } from '@/types/api';

interface AuthState {
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  agent: AgentProfile | null;

  hydrate: () => Promise<void>;
  login: (phone: string, otp: string) => Promise<void>;
  sendOTP: (phone: string) => Promise<void>;
  register: (params: RegisterAgentParams) => Promise<void>;
  fetchProfile: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  isAuthenticated: false,
  isLoading: true,
  agent: null,

  hydrate: async () => {
    try {
      const token = await getStoredToken();
      if (token) {
        set({ token, isAuthenticated: true });
        await get().fetchProfile();
      }
    } catch {
      await clearTokens();
    } finally {
      set({ isLoading: false });
    }
  },

  sendOTP: async (phone: string) => {
    await requestOTP(phone);
  },

  register: async (params: RegisterAgentParams) => {
    await registerAgent(params);
  },

  login: async (phone: string, otp: string) => {
    const result = await verifyOTP(phone, otp);
    set({ token: result.access_token, isAuthenticated: true });
    await get().fetchProfile();
  },

  fetchProfile: async () => {
    try {
      const { data } = await api.get<ApiResponse<AgentProfile>>('/v1/agents/me');
      set({ agent: data.data ?? null });
    } catch {
      // Profile fetch failed — not critical, will retry
    }
  },

  logout: async () => {
    await clearTokens();
    set({ token: null, isAuthenticated: false, agent: null });
  },
}));
