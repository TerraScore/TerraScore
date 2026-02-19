"use client";

import { useRouter } from "next/navigation";
import { useAlerts, useMarkAlertRead, useMarkAllAlertsRead } from "@/hooks/useAlerts";
import type { Alert } from "@/hooks/useAlerts";

const ALERT_CONFIG: Record<string, { icon: string; bg: string; iconBg: string; text: string; border: string }> = {
  "survey.submitted": {
    icon: "M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z",
    bg: "bg-green-50",
    iconBg: "bg-green-100 text-green-600",
    text: "text-green-700",
    border: "border-green-200",
  },
  "qa.completed": {
    icon: "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4",
    bg: "bg-blue-50",
    iconBg: "bg-blue-100 text-blue-600",
    text: "text-blue-700",
    border: "border-blue-200",
  },
  "report.generated": {
    icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
    bg: "bg-purple-50",
    iconBg: "bg-purple-100 text-purple-600",
    text: "text-purple-700",
    border: "border-purple-200",
  },
  "job.assigned": {
    icon: "M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z M15 11a3 3 0 11-6 0 3 3 0 016 0z",
    bg: "bg-indigo-50",
    iconBg: "bg-indigo-100 text-indigo-600",
    text: "text-indigo-700",
    border: "border-indigo-200",
  },
  "risk.high": {
    icon: "M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z",
    bg: "bg-red-50",
    iconBg: "bg-red-100 text-red-600",
    text: "text-red-700",
    border: "border-red-200",
  },
  "encroachment.detected": {
    icon: "M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z",
    bg: "bg-orange-50",
    iconBg: "bg-orange-100 text-orange-600",
    text: "text-orange-700",
    border: "border-orange-200",
  },
};

const DEFAULT_CONFIG = {
  icon: "M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9",
  bg: "bg-gray-50",
  iconBg: "bg-gray-100 text-gray-500",
  text: "text-gray-700",
  border: "border-gray-200",
};

function getAlertConfig(type: string) {
  return ALERT_CONFIG[type] ?? DEFAULT_CONFIG;
}

function getAlertLabel(type: string): string {
  const labels: Record<string, string> = {
    "survey.submitted": "Survey Completed",
    "qa.completed": "QA Review",
    "report.generated": "Report Ready",
    "job.assigned": "Agent Assigned",
    "risk.high": "Risk Alert",
    "encroachment.detected": "Encroachment",
  };
  return labels[type] ?? "Notification";
}

function getAlertLink(alert: Alert): string | null {
  const parcelId = alert.data?.parcel_id;
  if (!parcelId) return null;

  switch (alert.type) {
    case "survey.submitted":
    case "qa.completed":
      return `/parcels/${parcelId}/surveys`;
    case "report.generated":
      return `/parcels/${parcelId}`;
    case "job.assigned":
      return `/parcels/${parcelId}`;
    case "risk.high":
    case "encroachment.detected":
      return `/parcels/${parcelId}`;
    default:
      return parcelId ? `/parcels/${parcelId}` : null;
  }
}

function timeAgo(dateStr: string): string {
  const now = new Date();
  const d = new Date(dateStr);
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "Just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay}d ago`;
  return d.toLocaleDateString("en-IN", { day: "numeric", month: "short" });
}

export default function AlertsPage() {
  const router = useRouter();
  const { data, isLoading, error } = useAlerts();
  const markRead = useMarkAlertRead();
  const markAllRead = useMarkAllAlertsRead();

  const alerts: Alert[] = Array.isArray(data) ? data : [];
  const hasUnread = alerts.some((a) => !a.is_read);

  if (isLoading) {
    return (
      <div className="p-6">
        <h1 className="text-xl font-semibold text-gray-900 mb-4">Alerts</h1>
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-4 animate-pulse">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-gray-200 rounded-full" />
                <div className="flex-1">
                  <div className="h-4 bg-gray-200 rounded w-1/3 mb-2" />
                  <div className="h-3 bg-gray-100 rounded w-2/3" />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <h1 className="text-xl font-semibold text-gray-900 mb-4">Alerts</h1>
        <div className="bg-red-50 text-red-700 p-4 rounded-xl text-sm">
          Failed to load alerts. Please try again.
        </div>
      </div>
    );
  }

  function handleAlertClick(alert: Alert) {
    if (!alert.is_read) {
      markRead.mutate(alert.id);
    }
    const link = getAlertLink(alert);
    if (link) {
      router.push(link);
    }
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-xl font-semibold text-gray-900">Alerts</h1>
        {hasUnread && (
          <button
            onClick={() => markAllRead.mutate()}
            disabled={markAllRead.isPending}
            className="text-sm text-emerald-600 hover:text-emerald-700 font-medium disabled:opacity-50"
          >
            Mark all as read
          </button>
        )}
      </div>

      {alerts.length === 0 ? (
        <div className="text-center py-16">
          <div className="w-16 h-16 mx-auto bg-gray-100 rounded-full flex items-center justify-center mb-4">
            <svg className="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
          </div>
          <p className="text-gray-500 text-sm font-medium">No alerts yet</p>
          <p className="text-gray-400 text-xs mt-1">You&apos;ll be notified about surveys, reports, and risk alerts.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {alerts.map((alert: Alert) => {
            const config = getAlertConfig(alert.type);
            const link = getAlertLink(alert);
            const isClickable = !!link;

            return (
              <div
                key={alert.id}
                onClick={() => handleAlertClick(alert)}
                className={`rounded-xl border p-4 transition-all ${
                  alert.is_read
                    ? "bg-white border-gray-200"
                    : `${config.bg} ${config.border}`
                } ${isClickable ? "cursor-pointer hover:shadow-sm" : ""}`}
              >
                <div className="flex items-start gap-3">
                  {/* Icon */}
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${
                    alert.is_read ? "bg-gray-100 text-gray-400" : config.iconBg
                  }`}>
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d={config.icon} />
                    </svg>
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      {!alert.is_read && (
                        <span className="w-2 h-2 bg-emerald-500 rounded-full flex-shrink-0" />
                      )}
                      <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                        alert.is_read ? "bg-gray-100 text-gray-500" : `${config.iconBg}`
                      }`}>
                        {getAlertLabel(alert.type)}
                      </span>
                      <span className="text-xs text-gray-400 ml-auto flex-shrink-0">
                        {timeAgo(alert.created_at)}
                      </span>
                    </div>
                    <h3 className="text-sm font-medium text-gray-900 mt-1">
                      {alert.title}
                    </h3>
                    {alert.body && (
                      <p className="text-sm text-gray-600 mt-0.5 line-clamp-2">{alert.body}</p>
                    )}
                    {isClickable && (
                      <p className="text-xs text-emerald-600 font-medium mt-1.5">
                        View details &rarr;
                      </p>
                    )}
                  </div>

                  {/* Mark read button */}
                  {!alert.is_read && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        markRead.mutate(alert.id);
                      }}
                      disabled={markRead.isPending}
                      className="text-xs text-gray-400 hover:text-gray-600 flex-shrink-0 mt-1"
                    >
                      Mark read
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
