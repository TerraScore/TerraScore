"use client";

import { useAlerts, useMarkAlertRead, useMarkAllAlertsRead } from "@/hooks/useAlerts";
import type { Alert } from "@/hooks/useAlerts";

export default function AlertsPage() {
  const { data, isLoading, error } = useAlerts();
  const markRead = useMarkAlertRead();
  const markAllRead = useMarkAllAlertsRead();

  const alerts = data?.data ?? [];
  const hasUnread = alerts.some((a) => !a.is_read);

  if (isLoading) {
    return (
      <div className="p-6">
        <h1 className="text-xl font-semibold text-gray-900 mb-4">Alerts</h1>
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="bg-white rounded-lg border border-gray-200 p-4 animate-pulse">
              <div className="h-4 bg-gray-200 rounded w-1/3 mb-2" />
              <div className="h-3 bg-gray-100 rounded w-2/3" />
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
        <div className="bg-red-50 text-red-700 p-4 rounded-lg text-sm">
          Failed to load alerts. Please try again.
        </div>
      </div>
    );
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
        <div className="text-center py-12">
          <svg className="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
          </svg>
          <p className="text-gray-500 text-sm">No alerts yet</p>
          <p className="text-gray-400 text-xs mt-1">You&apos;ll be notified about survey reports and status changes.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {alerts.map((alert: Alert) => (
            <div
              key={alert.id}
              className={`bg-white rounded-lg border p-4 transition-colors ${
                alert.is_read
                  ? "border-gray-200"
                  : "border-emerald-200 bg-emerald-50/30"
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    {!alert.is_read && (
                      <span className="w-2 h-2 bg-emerald-500 rounded-full flex-shrink-0" />
                    )}
                    <h3 className="text-sm font-medium text-gray-900 truncate">
                      {alert.title}
                    </h3>
                  </div>
                  {alert.body && (
                    <p className="text-sm text-gray-600 mt-1">{alert.body}</p>
                  )}
                  <p className="text-xs text-gray-400 mt-1">
                    {new Date(alert.created_at).toLocaleString()}
                  </p>
                </div>
                {!alert.is_read && (
                  <button
                    onClick={() => markRead.mutate(alert.id)}
                    disabled={markRead.isPending}
                    className="text-xs text-gray-400 hover:text-gray-600 ml-2 flex-shrink-0"
                  >
                    Mark read
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
