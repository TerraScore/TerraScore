import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import type { UpdateBoundaryRequest } from "@/lib/types";

export function useUpdateBoundary(parcelId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateBoundaryRequest) =>
      apiClient.put(`/api/parcels/${parcelId}/boundary`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["parcel", parcelId] });
      queryClient.invalidateQueries({ queryKey: ["parcels"] });
    },
  });
}
