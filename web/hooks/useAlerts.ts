import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export interface Alert {
  id: string;
  type: string;
  title: string;
  body: string | null;
  data?: Record<string, string>;
  is_read: boolean;
  created_at: string;
}

export function useAlerts(page = 1) {
  return useQuery({
    queryKey: ["alerts", page],
    queryFn: () =>
      apiClient.get<Alert[]>(`/api/alerts?page=${page}&per_page=20`),
    refetchInterval: 30000, // Poll every 30s for new alerts
  });
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ["alerts", "unread-count"],
    queryFn: () =>
      apiClient.get<{ unread_count: number }>("/api/alerts/unread/count"),
    refetchInterval: 30000,
    select: (data) => data.unread_count,
  });
}

export function useMarkAlertRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (alertId: string) =>
      apiClient.put(`/api/alerts/${alertId}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });
}

export function useMarkAllAlertsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiClient.put("/api/alerts/read-all"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });
}
