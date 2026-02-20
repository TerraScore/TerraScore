import { useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
} from 'react-native';
import { useRouter } from 'expo-router';
import { useFocusEffect } from '@react-navigation/native';
import { api } from '@/services/api';
import { onAgentEvent } from '@/hooks/useJobOffers';
import type { Job, ApiResponse } from '@/types/api';

const ACTIVE_STATUSES = ['assigned', 'agent_arrived', 'survey_in_progress', 'agent_en_route'];

export default function ActiveJobsScreen() {
  const router = useRouter();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchJobs = useCallback(async () => {
    try {
      const { data } = await api.get<ApiResponse<Job[]>>('/v1/agents/me/jobs');
      const active = (data.data ?? []).filter((j) =>
        ACTIVE_STATUSES.includes(j.status ?? ''),
      );
      setJobs(active);
    } catch {
      // silently fail, user can pull to refresh
    } finally {
      setLoading(false);
    }
  }, []);

  // Refetch when screen comes into focus
  useFocusEffect(
    useCallback(() => {
      fetchJobs();
    }, [fetchJobs]),
  );

  // Listen for real-time WebSocket events
  useEffect(() => {
    const unsub = onAgentEvent((eventType) => {
      if (
        eventType === 'job.accepted' ||
        eventType === 'job.arrived' ||
        eventType === 'job.survey_submitted'
      ) {
        fetchJobs();
      }
    });
    return unsub;
  }, [fetchJobs]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    await fetchJobs();
    setRefreshing(false);
  }, [fetchJobs]);

  const renderJob = ({ item }: { item: Job }) => (
    <TouchableOpacity
      style={styles.card}
      onPress={() => router.push(`/job/${item.id}/details`)}
      activeOpacity={0.7}
    >
      <View style={styles.cardHeader}>
        <Text style={styles.surveyType}>{item.survey_type.replace(/_/g, ' ')}</Text>
        <View style={[styles.statusBadge, statusColor(item.status)]}>
          <Text style={styles.statusText}>{formatStatus(item.status)}</Text>
        </View>
      </View>
      <Text style={styles.deadline}>
        Deadline: {new Date(item.deadline).toLocaleDateString()}
      </Text>
      {item.parcel && (
        <Text style={styles.location}>
          {[item.parcel.village, item.parcel.district, item.parcel.state]
            .filter(Boolean)
            .join(', ')}
        </Text>
      )}
    </TouchableOpacity>
  );

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563eb" />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      {jobs.length === 0 ? (
        <View style={styles.center}>
          <Text style={styles.emptyIcon}>📋</Text>
          <Text style={styles.emptyTitle}>No active jobs</Text>
          <Text style={styles.emptySubtitle}>
            Accept an offer to start a field survey
          </Text>
        </View>
      ) : (
        <FlatList
          data={jobs}
          keyExtractor={(item) => item.id}
          renderItem={renderJob}
          contentContainerStyle={styles.list}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
          }
        />
      )}
    </View>
  );
}

function formatStatus(status: string | null | undefined): string {
  switch (status) {
    case 'assigned':
      return 'Assigned';
    case 'agent_en_route':
      return 'En Route';
    case 'agent_arrived':
      return 'Arrived';
    case 'survey_in_progress':
      return 'In Progress';
    default:
      return status ?? 'Unknown';
  }
}

function statusColor(status: string | null | undefined) {
  switch (status) {
    case 'assigned':
      return { backgroundColor: '#dbeafe' };
    case 'agent_en_route':
      return { backgroundColor: '#e0e7ff' };
    case 'agent_arrived':
      return { backgroundColor: '#dcfce7' };
    case 'survey_in_progress':
      return { backgroundColor: '#fef3c7' };
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
    paddingHorizontal: 32,
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#333',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#666',
    textAlign: 'center',
  },
  list: {
    padding: 16,
  },
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 2,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  surveyType: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1a1a2e',
    textTransform: 'capitalize',
  },
  statusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 8,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '500',
    color: '#333',
  },
  deadline: {
    fontSize: 13,
    color: '#666',
    marginBottom: 4,
  },
  location: {
    fontSize: 13,
    color: '#888',
  },
});
