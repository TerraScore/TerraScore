import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ApiResponse, Parcel } from "@/lib/types";

export function useParcel(id: string | null) {
  return useQuery({
    queryKey: ["parcel", id],
    queryFn: () => apiClient.get<ApiResponse<Parcel>>(`/api/parcels/${id}`),
    enabled: !!id,
  });
}
