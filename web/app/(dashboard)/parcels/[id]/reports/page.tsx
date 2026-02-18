import type { Metadata } from "next";
import { ComingSoon } from "@/components/ui/ComingSoon";

export const metadata: Metadata = { title: "Reports — TerraScore" };

export default function ReportsPage() {
  return <ComingSoon title="Reports" description="Verification reports and land assessment documents will be available here." />;
}
