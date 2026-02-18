import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { haversineDistance } from '@/hooks/useLocation';
import { api } from '@/services/api';

interface LocationState {
  lat: number;
  lng: number;
  accuracy: number;
}

interface Props {
  jobId: string;
  targetLat: number;
  targetLng: number;
  location: LocationState | null;
  permissionGranted: boolean;
  locationError: string | null;
  onArrived: () => void;
}

const ARRIVAL_THRESHOLD_M = 500;

export function GeofenceGate({
  jobId,
  targetLat,
  targetLng,
  location,
  permissionGranted,
  locationError,
  onArrived,
}: Props) {
  const [confirming, setConfirming] = useState(false);

  const distance = location
    ? haversineDistance(location.lat, location.lng, targetLat, targetLng)
    : null;

  const withinRange = distance !== null && distance <= ARRIVAL_THRESHOLD_M;

  const handleConfirmArrival = async () => {
    if (!location) return;

    setConfirming(true);
    try {
      await api.post(`/v1/jobs/${jobId}/arrive`, {
        lat: location.lat,
        lng: location.lng,
        accuracy: location.accuracy,
      });
      onArrived();
    } catch (err: any) {
      const msg =
        err?.response?.data?.error?.message ??
        'Server rejected arrival. Move closer to the parcel.';
      Alert.alert('Arrival Failed', msg);
    } finally {
      setConfirming(false);
    }
  };

  if (locationError) {
    return (
      <View style={styles.container}>
        <Text style={styles.errorText}>{locationError}</Text>
      </View>
    );
  }

  if (!permissionGranted || !location) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color="#2563eb" />
        <Text style={styles.loadingText}>Getting your location...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.distanceLabel}>Distance to parcel</Text>
      <Text style={styles.distance}>
        {distance! < 1000
          ? `${Math.round(distance!)} m`
          : `${(distance! / 1000).toFixed(1)} km`}
      </Text>

      {withinRange ? (
        <TouchableOpacity
          style={styles.arriveButton}
          onPress={handleConfirmArrival}
          disabled={confirming}
        >
          {confirming ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.arriveText}>Confirm Arrival</Text>
          )}
        </TouchableOpacity>
      ) : (
        <Text style={styles.hint}>
          Move within {ARRIVAL_THRESHOLD_M}m of the parcel to confirm arrival
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    padding: 24,
  },
  distanceLabel: {
    fontSize: 14,
    color: '#666',
    marginBottom: 4,
  },
  distance: {
    fontSize: 36,
    fontWeight: '700',
    color: '#1a1a2e',
    marginBottom: 20,
  },
  arriveButton: {
    backgroundColor: '#22c55e',
    paddingVertical: 16,
    paddingHorizontal: 48,
    borderRadius: 12,
  },
  arriveText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  hint: {
    fontSize: 13,
    color: '#888',
    textAlign: 'center',
  },
  errorText: {
    fontSize: 14,
    color: '#ef4444',
    textAlign: 'center',
  },
  loadingText: {
    fontSize: 14,
    color: '#666',
    marginTop: 12,
  },
});
