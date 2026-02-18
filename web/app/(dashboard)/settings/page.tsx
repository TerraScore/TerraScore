import type { Metadata } from "next";
import { ComingSoon } from "@/components/ui/ComingSoon";

export const metadata: Metadata = { title: "Settings — TerraScore" };

export default function SettingsPage() {
  return <ComingSoon title="Settings" description="Account settings, notification preferences, and profile management." />;
}
