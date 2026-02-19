import { api, storeTokens, clearTokens, getStoredToken, getStoredRefreshToken } from './api';
import type { ApiResponse, VerifyOTPResponse } from '@/types/api';

// Re-export for convenience so other modules can import from either place
export { getStoredToken, clearTokens, getStoredRefreshToken };

export async function requestOTP(phone: string): Promise<void> {
  await api.post('/v1/auth/login', { phone });
}

export async function verifyOTP(phone: string, otp: string): Promise<VerifyOTPResponse> {
  const { data } = await api.post<ApiResponse<VerifyOTPResponse>>('/v1/auth/verify-otp', {
    phone,
    otp,
  });
  const result = data.data!;
  await storeTokens(result.access_token, result.refresh_token);
  return result;
}

export interface RegisterAgentParams {
  phone: string;
  full_name: string;
  email?: string;
  home_lat?: number;
  home_lng?: number;
  state_code?: string;
  district_code?: string;
}

export async function registerAgent(params: RegisterAgentParams): Promise<void> {
  await api.post('/v1/agents/register', params);
}
