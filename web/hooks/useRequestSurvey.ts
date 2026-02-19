import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export function useRequestSurvey() {
  return useMutation({
    mutationFn: (parcelId: string) =>
      apiClient.post(`/api/parcels/${parcelId}/request-survey`),
  });
}
