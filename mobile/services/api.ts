import axios from 'axios';
import * as SecureStore from 'expo-secure-store';

const ACCESS_TOKEN_KEY = 'terrascore_access_token';
const REFRESH_TOKEN_KEY = 'terrascore_refresh_token';

export const BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';

export const api = axios.create({
  baseURL: BASE_URL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

// --- Token helpers (kept here to avoid circular deps with auth.ts) ---

export async function getStoredToken(): Promise<string | null> {
  return SecureStore.getItemAsync(ACCESS_TOKEN_KEY);
}

export async function getStoredRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
}

export async function storeTokens(accessToken: string, refreshToken: string): Promise<void> {
  await SecureStore.setItemAsync(ACCESS_TOKEN_KEY, accessToken);
  await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refreshToken);
}

export async function clearTokens(): Promise<void> {
  await SecureStore.deleteItemAsync(ACCESS_TOKEN_KEY);
  await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
}

// --- Interceptors ---

// Attach JWT to every request
api.interceptors.request.use(async (config) => {
  const token = await getStoredToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Auto-refresh on 401
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string | null) => void;
  reject: (err: unknown) => void;
}> = [];

function processQueue(token: string | null) {
  failedQueue.forEach(({ resolve }) => resolve(token));
  failedQueue = [];
}

/**
 * Refresh using a plain axios call (NOT the `api` instance) to avoid interceptor loops.
 */
async function doRefresh(): Promise<string | null> {
  const refresh = await getStoredRefreshToken();
  if (!refresh) return null;

  try {
    const { data } = await axios.post(`${BASE_URL}/v1/auth/refresh`, {
      refresh_token: refresh,
    });
    const result = data.data;
    if (!result?.access_token) return null;
    await storeTokens(result.access_token, result.refresh_token);
    return result.access_token;
  } catch {
    await clearTokens();
    return null;
  }
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      }).then((token) => {
        if (token) {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return api(originalRequest);
        }
        return Promise.reject(error);
      });
    }

    originalRequest._retry = true;
    isRefreshing = true;

    try {
      const newToken = await doRefresh();
      isRefreshing = false;
      processQueue(newToken);

      if (newToken) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      }

      return Promise.reject(error);
    } catch (refreshError) {
      isRefreshing = false;
      processQueue(null);
      return Promise.reject(refreshError);
    }
  },
);
