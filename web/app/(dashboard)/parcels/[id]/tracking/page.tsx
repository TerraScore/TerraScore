import type { Metadata } from "next";
import { ComingSoon } from "@/components/ui/ComingSoon";

export const metadata: Metadata = { title: "Live Tracking — TerraScore" };

export default function TrackingPage() {
  return <ComingSoon title="Live Tracking" description="Real-time agent tracking and GPS trail visualization will be available here." />;
}
