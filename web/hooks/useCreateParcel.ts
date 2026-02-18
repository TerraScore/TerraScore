import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { ApiResponse, Parcel, CreateParcelRequest } from "@/lib/types";

export function useCreateParcel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateParcelRequest) =>
      apiClient.post<ApiResponse<Parcel>>("/api/parcels", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["parcels"] });
    },
  });
}
