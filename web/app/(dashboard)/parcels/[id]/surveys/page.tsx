"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { apiClient } from "@/lib/api-client";
import { Spinner } from "@/components/ui/Spinner";

interface SurveyJob {
  id: string;
  survey_type: string;
  status: string | null;
  qa_score: number | null;
  qa_status: string | null;
  qa_notes: string | null;
  responses: Record<string, string> | null;
  created_at: string;
  completed_at: string | null;
}

export default function SurveysPage() {
  const params = useParams<{ id: string }>();
  const [surveys, setSurveys] = useState<SurveyJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const body = await apiClient.get<{ data: SurveyJob[] }>(`/api/parcels/${params.id}/surveys`);
        setSurveys(body.data ?? []);
      } catch {
        setError("Failed to load surveys");
      } finally {
        setLoading(false);
      }
    })();
  }, [params.id]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner className="h-8 w-8" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <p className="text-red-600">{error}</p>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-4">
        <Link href={`/parcels/${params.id}`} className="text-sm text-gray-500 hover:text-gray-700">
          &larr; Back to parcel
        </Link>
      </div>

      <h1 className="text-lg font-bold text-gray-900 mb-4">Field Surveys</h1>

      {surveys.length === 0 ? (
        <div className="bg-white rounded-xl border border-gray-200 p-8 text-center">
          <p className="text-gray-500">No surveys yet. Request a field survey from the parcel detail page.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {surveys.map((s) => (
            <div key={s.id} className="bg-white rounded-xl border border-gray-200 p-5">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <span className="text-sm font-medium text-gray-900 capitalize">
                    {s.survey_type.replace(/_/g, " ")}
                  </span>
                  <p className="text-xs text-gray-500 mt-0.5">
                    {new Date(s.created_at).toLocaleDateString("en-IN", {
                      day: "numeric",
                      month: "short",
                      year: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </p>
                </div>
                <StatusBadge status={s.status} />
              </div>

              {s.qa_score != null && (
                <div className="mb-3">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs text-gray-500">QA Score</span>
                    <QABadge status={s.qa_status} />
                  </div>
                  <div className="w-full bg-gray-100 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${
                        s.qa_score >= 0.7 ? "bg-green-500" : s.qa_score >= 0.4 ? "bg-yellow-500" : "bg-red-500"
                      }`}
                      style={{ width: `${Math.round(s.qa_score * 100)}%` }}
                    />
                  </div>
                  <p className="text-xs text-gray-500 mt-1">{Math.round(s.qa_score * 100)}%</p>
                </div>
              )}

              {s.qa_notes && (
                <p className="text-xs text-gray-600 bg-gray-50 rounded-lg p-2 mb-3">{s.qa_notes}</p>
              )}

              {s.responses && Object.keys(s.responses).length > 0 && (
                <div className="border-t border-gray-100 pt-3">
                  <p className="text-xs font-medium text-gray-500 mb-2">Responses</p>
                  <div className="space-y-1">
                    {Object.entries(s.responses).map(([key, value]) => (
                      <div key={key} className="flex justify-between text-xs">
                        <span className="text-gray-600 capitalize">{key.replace(/_/g, " ")}</span>
                        <ResponseValue value={value} />
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string | null }) {
  const s = status ?? "pending";
  const colors: Record<string, string> = {
    pending: "bg-gray-100 text-gray-700",
    dispatching: "bg-blue-100 text-blue-700",
    assigned: "bg-blue-100 text-blue-700",
    agent_en_route: "bg-indigo-100 text-indigo-700",
    agent_arrived: "bg-indigo-100 text-indigo-700",
    survey_submitted: "bg-yellow-100 text-yellow-700",
    completed: "bg-green-100 text-green-700",
    cancelled: "bg-red-100 text-red-700",
  };
  return (
    <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${colors[s] ?? "bg-gray-100 text-gray-700"}`}>
      {s.replace(/_/g, " ")}
    </span>
  );
}

function QABadge({ status }: { status: string | null }) {
  if (!status) return null;
  const colors: Record<string, string> = {
    passed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    flagged: "bg-yellow-100 text-yellow-700",
  };
  return (
    <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${colors[status] ?? "bg-gray-100 text-gray-700"}`}>
      {status}
    </span>
  );
}

function ResponseValue({ value }: { value: string }) {
  if (value === "yes") return <span className="text-green-600 font-medium">Yes</span>;
  if (value === "no") return <span className="text-red-600 font-medium">No</span>;
  if (value === "na") return <span className="text-gray-400">N/A</span>;
  return <span className="text-gray-700">{value}</span>;
}
