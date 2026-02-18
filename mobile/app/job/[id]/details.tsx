import { useEffect, useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { api } from '@/services/api';
import type { Job, ApiResponse } from '@/types/api';

export default function JobDetailsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get<ApiResponse<Job>>(`/v1/jobs/${id}`);
        setJob(data.data ?? null);
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    })();
  }, [id]);

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563eb" />
      </View>
    );
  }

  if (!job) {
    return (
      <View style={styles.center}>
        <Text style={styles.errorText}>Job not found</Text>
      </View>
    );
  }

  const canNavigate = job.status === 'assigned';
  const canSurvey = job.status === 'agent_arrived' || job.status === 'survey_in_progress';

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Text style={styles.backButton}>← Back</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.card}>
        <Text style={styles.title}>
          {job.survey_type.replace(/_/g, ' ')}
        </Text>
        <View style={[styles.statusBadge, getStatusStyle(job.status)]}>
          <Text style={styles.statusText}>{formatStatus(job.status)}</Text>
        </View>
      </View>

      <View style={styles.card}>
        <Text style={styles.sectionTitle}>Details</Text>
        <InfoRow label="Deadline" value={new Date(job.deadline).toLocaleDateString()} />
        {job.priority && <InfoRow label="Priority" value={job.priority} />}
        <InfoRow label="Created" value={new Date(job.created_at).toLocaleDateString()} />
      </View>

      {job.parcel && (
        <View style={styles.card}>
          <Text style={styles.sectionTitle}>Parcel Info</Text>
          {job.parcel.label && <InfoRow label="Label" value={job.parcel.label} />}
          {job.parcel.survey_number && (
            <InfoRow label="Survey No." value={job.parcel.survey_number} />
          )}
          {job.parcel.village && <InfoRow label="Village" value={job.parcel.village} />}
          <InfoRow label="District" value={job.parcel.district} />
          <InfoRow label="State" value={job.parcel.state} />
          {job.parcel.land_type && <InfoRow label="Land Type" value={job.parcel.land_type} />}
          {job.parcel.area_sqm && (
            <InfoRow label="Area" value={`${job.parcel.area_sqm.toLocaleString()} sq m`} />
          )}
        </View>
      )}

      <View style={styles.actions}>
        {canNavigate && (
          <TouchableOpacity
            style={[styles.actionButton, styles.navigateButton]}
            onPress={() => router.push(`/job/${id}/navigate`)}
          >
            <Text style={styles.actionButtonText}>Navigate to Parcel</Text>
          </TouchableOpacity>
        )}
        {canSurvey && (
          <TouchableOpacity
            style={[styles.actionButton, styles.surveyButton]}
            onPress={() => router.push(`/job/${id}/survey`)}
          >
            <Text style={styles.actionButtonText}>Start Survey</Text>
          </TouchableOpacity>
        )}
      </View>

      <View style={{ height: 40 }} />
    </ScrollView>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{label}</Text>
      <Text style={styles.infoValue}>{value}</Text>
    </View>
  );
}

function formatStatus(status: string | null | undefined): string {
  return (status ?? 'unknown').replace(/_/g, ' ');
}

function getStatusStyle(status: string | null | undefined) {
  switch (status) {
    case 'assigned':
      return { backgroundColor: '#dbeafe' };
    case 'agent_arrived':
      return { backgroundColor: '#dcfce7' };
    case 'survey_in_progress':
      return { backgroundColor: '#fef3c7' };
    case 'survey_submitted':
      return { backgroundColor: '#d1fae5' };
    default:
      return { backgroundColor: '#f3f4f6' };
  }
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  errorText: {
    fontSize: 16,
    color: '#ef4444',
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 60,
    paddingBottom: 8,
  },
  backButton: {
    fontSize: 16,
    color: '#2563eb',
    fontWeight: '500',
  },
  card: {
    backgroundColor: '#fff',
    marginHorizontal: 16,
    marginTop: 12,
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 2,
  },
  title: {
    fontSize: 22,
    fontWeight: '700',
    color: '#1a1a2e',
    textTransform: 'capitalize',
    marginBottom: 8,
  },
  statusBadge: {
    alignSelf: 'flex-start',
    paddingHorizontal: 12,
    paddingVertical: 5,
    borderRadius: 8,
  },
  statusText: {
    fontSize: 13,
    fontWeight: '500',
    color: '#333',
    textTransform: 'capitalize',
  },
  sectionTitle: {
    fontSize: 15,
    fontWeight: '600',
    color: '#666',
    marginBottom: 12,
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 6,
    borderBottomWidth: 1,
    borderBottomColor: '#f3f4f6',
  },
  infoLabel: {
    fontSize: 14,
    color: '#888',
  },
  infoValue: {
    fontSize: 14,
    color: '#333',
    fontWeight: '500',
  },
  actions: {
    paddingHorizontal: 16,
    marginTop: 20,
    gap: 12,
  },
  actionButton: {
    paddingVertical: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  navigateButton: {
    backgroundColor: '#2563eb',
  },
  surveyButton: {
    backgroundColor: '#22c55e',
  },
  actionButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
