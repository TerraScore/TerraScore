import { useEffect, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Platform,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';
import * as Linking from 'expo-linking';
import { api } from '@/services/api';
import { useLocation, haversineDistance } from '@/hooks/useLocation';
import { GeofenceGate } from '@/components/survey/GeofenceGate';
import type { Job, ApiResponse } from '@/types/api';

export default function NavigateScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { location, permissionGranted, error: locationError } = useLocation(15000);
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [arrived, setArrived] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get<ApiResponse<Job>>(`/v1/jobs/${id}`);
        setJob(data.data ?? null);
        if (data.data?.status === 'agent_arrived') {
          setArrived(true);
        }
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    })();
  }, [id]);

  const parcelCenter = getParcelCenter(job);

  const openGoogleMaps = () => {
    if (!parcelCenter) {
      Alert.alert('Error', 'Parcel location not available.');
      return;
    }

    const { lat, lng } = parcelCenter;
    const url = Platform.select({
      ios: `comgooglemaps://?daddr=${lat},${lng}&directionsmode=driving`,
      android: `google.navigation:q=${lat},${lng}`,
    });

    const fallbackUrl = `https://www.google.com/maps/dir/?api=1&destination=${lat},${lng}`;

    if (url) {
      Linking.canOpenURL(url).then((supported) => {
        Linking.openURL(supported ? url : fallbackUrl);
      });
    } else {
      Linking.openURL(fallbackUrl);
    }
  };

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563eb" />
      </View>
    );
  }

  if (!job || !parcelCenter) {
    return (
      <View style={styles.center}>
        <Text style={styles.errorText}>Job or parcel location not found</Text>
      </View>
    );
  }

  const distance = location
    ? haversineDistance(location.lat, location.lng, parcelCenter.lat, parcelCenter.lng)
    : null;

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <TouchableOpacity onPress={() => router.back()}>
          <Text style={styles.backButton}>← Back</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Navigate to Parcel</Text>
      </View>

      {arrived ? (
        <View style={styles.arrivedContainer}>
          <Text style={styles.arrivedIcon}>✓</Text>
          <Text style={styles.arrivedTitle}>Arrived at parcel</Text>
          <Text style={styles.arrivedSubtitle}>You can now start the survey</Text>
          <TouchableOpacity
            style={styles.surveyButton}
            onPress={() => router.replace(`/job/${id}/survey`)}
          >
            <Text style={styles.surveyButtonText}>Start Survey</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <>
          {distance != null && (
            <View style={styles.distanceCard}>
              <Text style={styles.distanceLabel}>Distance</Text>
              <Text style={styles.distanceValue}>
                {distance < 1000
                  ? `${Math.round(distance)} m`
                  : `${(distance / 1000).toFixed(1)} km`}
              </Text>
            </View>
          )}

          <TouchableOpacity style={styles.mapsButton} onPress={openGoogleMaps}>
            <Text style={styles.mapsButtonText}>Open in Google Maps</Text>
          </TouchableOpacity>

          <GeofenceGate
            jobId={id!}
            targetLat={parcelCenter.lat}
            targetLng={parcelCenter.lng}
            location={location}
            permissionGranted={permissionGranted}
            locationError={locationError}
            onArrived={() => setArrived(true)}
          />
        </>
      )}
    </View>
  );
}

function getParcelCenter(job: Job | null): { lat: number; lng: number } | null {
  if (!job?.parcel?.boundary_geojson) return null;

  try {
    const geo =
      typeof job.parcel.boundary_geojson === 'string'
        ? JSON.parse(job.parcel.boundary_geojson)
        : job.parcel.boundary_geojson;

    // Try to extract centroid from GeoJSON polygon
    const coords =
      geo.type === 'Polygon'
        ? geo.coordinates[0]
        : geo.type === 'Feature' && geo.geometry?.type === 'Polygon'
          ? geo.geometry.coordinates[0]
          : null;

    if (!coords || coords.length === 0) return null;

    let latSum = 0;
    let lngSum = 0;
    for (const [lng, lat] of coords) {
      latSum += lat;
      lngSum += lng;
    }
    return {
      lat: latSum / coords.length,
      lng: lngSum / coords.length,
    };
  } catch {
    return null;
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
    paddingBottom: 12,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  backButton: {
    fontSize: 16,
    color: '#2563eb',
    fontWeight: '500',
    marginBottom: 4,
  },
  headerTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#1a1a2e',
  },
  distanceCard: {
    backgroundColor: '#fff',
    margin: 16,
    borderRadius: 12,
    padding: 24,
    alignItems: 'center',
  },
  distanceLabel: {
    fontSize: 14,
    color: '#666',
    marginBottom: 4,
  },
  distanceValue: {
    fontSize: 42,
    fontWeight: '700',
    color: '#1a1a2e',
  },
  mapsButton: {
    backgroundColor: '#2563eb',
    marginHorizontal: 16,
    paddingVertical: 16,
    borderRadius: 12,
    alignItems: 'center',
    marginBottom: 8,
  },
  mapsButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  arrivedContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 32,
  },
  arrivedIcon: {
    fontSize: 64,
    color: '#22c55e',
    marginBottom: 16,
  },
  arrivedTitle: {
    fontSize: 24,
    fontWeight: '700',
    color: '#1a1a2e',
    marginBottom: 8,
  },
  arrivedSubtitle: {
    fontSize: 15,
    color: '#666',
    marginBottom: 32,
  },
  surveyButton: {
    backgroundColor: '#22c55e',
    paddingVertical: 16,
    paddingHorizontal: 48,
    borderRadius: 12,
  },
  surveyButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
