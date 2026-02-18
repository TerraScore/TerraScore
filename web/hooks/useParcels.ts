import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ApiResponse, Parcel } from "@/lib/types";

export function useParcels(page = 1) {
  return useQuery({
    queryKey: ["parcels", page],
    queryFn: () => apiClient.get<ApiResponse<Parcel[]>>(`/api/parcels?page=${page}&per_page=20`),
  });
}
