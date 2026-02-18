import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useRouter } from 'expo-router';
import { api } from '@/services/api';
import type { Offer } from '@/types/api';

interface Props {
  offer: Offer;
  onRemove?: () => void;
}

export function JobCard({ offer, onRemove }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState<'accept' | 'decline' | null>(null);

  const timeLeft = getTimeLeft(offer.expires_at);

  const handleAccept = async () => {
    setLoading('accept');
    try {
      await api.post(`/v1/jobs/${offer.job_id}/accept`);
      router.push(`/job/${offer.job_id}/details`);
    } catch (err: any) {
      Alert.alert('Error', err?.response?.data?.error?.message ?? 'Failed to accept offer.');
    } finally {
      setLoading(null);
    }
  };

  const handleDecline = async () => {
    setLoading('decline');
    try {
      await api.post(`/v1/jobs/${offer.job_id}/decline`, {});
      onRemove?.();
    } catch (err: any) {
      Alert.alert('Error', err?.response?.data?.error?.message ?? 'Failed to decline offer.');
    } finally {
      setLoading(null);
    }
  };

  return (
    <View style={styles.card}>
      <View style={styles.header}>
        <Text style={styles.type}>
          {offer.job?.survey_type?.replace(/_/g, ' ') ?? 'Survey Job'}
        </Text>
        <Text style={[styles.timer, timeLeft.urgent && styles.timerUrgent]}>
          {timeLeft.label}
        </Text>
      </View>

      {offer.job?.parcel && (
        <Text style={styles.location}>
          {[offer.job.parcel.village, offer.job.parcel.district]
            .filter(Boolean)
            .join(', ')}
        </Text>
      )}

      <View style={styles.details}>
        {offer.distance_km != null && (
          <DetailChip label={`${offer.distance_km.toFixed(1)} km`} icon="📍" />
        )}
        {offer.job?.deadline && (
          <DetailChip
            label={`Due ${new Date(offer.job.deadline).toLocaleDateString()}`}
            icon="📅"
          />
        )}
        {offer.job?.priority && (
          <DetailChip
            label={offer.job.priority}
            icon={offer.job.priority === 'urgent' ? '🔴' : '🟡'}
          />
        )}
      </View>

      <View style={styles.actions}>
        <TouchableOpacity
          style={[styles.button, styles.declineButton]}
          onPress={handleDecline}
          disabled={loading !== null}
        >
          {loading === 'decline' ? (
            <ActivityIndicator size="small" color="#ef4444" />
          ) : (
            <Text style={styles.declineText}>Decline</Text>
          )}
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.button, styles.acceptButton]}
          onPress={handleAccept}
          disabled={loading !== null}
        >
          {loading === 'accept' ? (
            <ActivityIndicator size="small" color="#fff" />
          ) : (
            <Text style={styles.acceptText}>Accept</Text>
          )}
        </TouchableOpacity>
      </View>
    </View>
  );
}

function DetailChip({ label, icon }: { label: string; icon: string }) {
  return (
    <View style={styles.chip}>
      <Text style={styles.chipIcon}>{icon}</Text>
      <Text style={styles.chipText}>{label}</Text>
    </View>
  );
}

function getTimeLeft(expiresAt: string) {
  const diff = new Date(expiresAt).getTime() - Date.now();
  if (diff <= 0) return { label: 'Expired', urgent: true };
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return { label: '<1 min', urgent: true };
  if (mins < 5) return { label: `${mins}m left`, urgent: true };
  return { label: `${mins}m left`, urgent: false };
}

const styles = StyleSheet.create({
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
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  type: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1a1a2e',
    textTransform: 'capitalize',
  },
  timer: {
    fontSize: 12,
    fontWeight: '500',
    color: '#666',
    backgroundColor: '#f3f4f6',
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 6,
  },
  timerUrgent: {
    color: '#ef4444',
    backgroundColor: '#fef2f2',
  },
  location: {
    fontSize: 13,
    color: '#888',
    marginBottom: 10,
  },
  details: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 14,
  },
  chip: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#f8f9fa',
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: 8,
  },
  chipIcon: {
    fontSize: 12,
    marginRight: 4,
  },
  chipText: {
    fontSize: 12,
    color: '#555',
  },
  actions: {
    flexDirection: 'row',
    gap: 10,
  },
  button: {
    flex: 1,
    paddingVertical: 12,
    borderRadius: 10,
    alignItems: 'center',
  },
  declineButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#e5e7eb',
  },
  acceptButton: {
    backgroundColor: '#2563eb',
  },
  declineText: {
    color: '#666',
    fontWeight: '600',
    fontSize: 14,
  },
  acceptText: {
    color: '#fff',
    fontWeight: '600',
    fontSize: 14,
  },
});
