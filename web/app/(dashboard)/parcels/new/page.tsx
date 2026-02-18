import type { Metadata } from "next";
import Link from "next/link";
import { NewParcelForm } from "@/components/parcels/NewParcelForm";

export const metadata: Metadata = {
  title: "New Parcel — TerraScore",
};

export default function NewParcelPage() {
  return (
    <div className="max-w-3xl mx-auto p-6">
      <div className="mb-6">
        <Link href="/" className="text-sm text-gray-500 hover:text-gray-700">
          &larr; Back to Dashboard
        </Link>
        <h1 className="text-xl font-bold text-gray-900 mt-2">Register New Parcel</h1>
      </div>
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <NewParcelForm />
      </div>
    </div>
  );
}
