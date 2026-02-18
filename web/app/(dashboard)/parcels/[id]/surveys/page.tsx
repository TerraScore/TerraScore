import type { Metadata } from "next";
import { ComingSoon } from "@/components/ui/ComingSoon";

export const metadata: Metadata = { title: "Surveys — TerraScore" };

export default function SurveysPage() {
  return <ComingSoon title="Surveys" description="Field verification surveys and photo evidence will be available here." />;
}
