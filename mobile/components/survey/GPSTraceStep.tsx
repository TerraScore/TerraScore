import { useEffect, useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import * as Location from 'expo-location';
import type { SurveyStep } from '@/types/api';
import type { GPSPoint } from '@/stores/surveyStore';

interface Props {
  step: SurveyStep;
  points: GPSPoint[];
  onPointAdded: (point: GPSPoint) => void;
  onComplete: (geoJson: string) => void;
  completed: boolean;
}

const TRACK_INTERVAL_MS = 3000;

export function GPSTraceStep({ step, points, onPointAdded, onComplete, completed }: Props) {
  const [tracking, setTracking] = useState(false);
  const [currentPos, setCurrentPos] = useState<GPSPoint | null>(null);
  const watchRef = useRef<Location.LocationSubscription | null>(null);

  useEffect(() => {
    return () => {
      watchRef.current?.remove();
    };
  }, []);

  const startTracking = async () => {
    const { status } = await Location.requestForegroundPermissionsAsync();
    if (status !== 'granted') return;

    setTracking(true);

    watchRef.current = await Location.watchPositionAsync(
      {
        accuracy: Location.Accuracy.High,
        timeInterval: TRACK_INTERVAL_MS,
        distanceInterval: 2,
      },
      (pos) => {
        const point: GPSPoint = {
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy ?? 0,
          timestamp: pos.timestamp,
        };
        setCurrentPos(point);
        onPointAdded(point);
      },
    );
  };

  const stopTracking = () => {
    watchRef.current?.remove();
    watchRef.current = null;
    setTracking(false);

    if (points.length >= 2) {
      // Build GeoJSON LineString
      const coordinates = points.map((p) => [p.lng, p.lat]);
      const geoJson = JSON.stringify({
        type: 'LineString',
        coordinates,
      });
      onComplete(geoJson);
    }
  };

  if (completed) {
    return (
      <View style={styles.container}>
        <Text style={styles.title}>{step.title}</Text>
        <View style={styles.completedCard}>
          <Text style={styles.completedText}>
            Boundary recorded ({points.length} points)
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.title}>{step.title}</Text>
      {step.description && <Text style={styles.description}>{step.description}</Text>}

      {tracking ? (
        <View>
          <View style={styles.trackingCard}>
            <View style={styles.trackingBadge}>
              <View style={styles.trackingDot} />
              <Text style={styles.trackingText}>Recording boundary walk</Text>
            </View>
            <Text style={styles.pointCount}>{points.length} points recorded</Text>
            {currentPos && (
              <Text style={styles.coordsText}>
                {currentPos.lat.toFixed(6)}, {currentPos.lng.toFixed(6)}
              </Text>
            )}
          </View>
          <TouchableOpacity style={styles.stopButton} onPress={stopTracking}>
            <Text style={styles.stopText}>
              {points.length >= 2 ? 'Stop & Save Trace' : 'Stop (need 2+ points)'}
            </Text>
          </TouchableOpacity>
        </View>
      ) : (
        <View>
          <Text style={styles.instructions}>
            Walk along the boundary of the parcel. GPS points will be recorded automatically.
          </Text>
          <TouchableOpacity style={styles.startButton} onPress={startTracking}>
            <Text style={styles.startText}>Start Boundary Walk</Text>
          </TouchableOpacity>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
  },
  title: {
    fontSize: 17,
    fontWeight: '600',
    color: '#1a1a2e',
    marginBottom: 6,
  },
  description: {
    fontSize: 14,
    color: '#666',
    marginBottom: 16,
  },
  instructions: {
    fontSize: 14,
    color: '#666',
    marginBottom: 16,
    lineHeight: 20,
  },
  trackingCard: {
    backgroundColor: '#eff6ff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  trackingBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  trackingDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#22c55e',
    marginRight: 6,
  },
  trackingText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#2563eb',
  },
  pointCount: {
    fontSize: 24,
    fontWeight: '700',
    color: '#1a1a2e',
  },
  coordsText: {
    fontSize: 12,
    color: '#888',
    fontFamily: 'SpaceMono',
    marginTop: 4,
  },
  completedCard: {
    backgroundColor: '#dcfce7',
    borderRadius: 12,
    padding: 16,
  },
  completedText: {
    fontSize: 15,
    fontWeight: '500',
    color: '#16a34a',
  },
  startButton: {
    backgroundColor: '#2563eb',
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
  },
  startText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
  stopButton: {
    backgroundColor: '#ef4444',
    paddingVertical: 14,
    borderRadius: 10,
    alignItems: 'center',
  },
  stopText: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
});
