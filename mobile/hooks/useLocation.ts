import { useEffect, useRef, useState } from 'react';
import * as Location from 'expo-location';

interface LocationState {
  lat: number;
  lng: number;
  accuracy: number;
}

export function useLocation(trackInterval = 15000) {
  const [location, setLocation] = useState<LocationState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [permissionGranted, setPermissionGranted] = useState(false);
  const watchRef = useRef<Location.LocationSubscription | null>(null);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') {
        setError('Location permission is required for field surveys.');
        return;
      }
      if (cancelled) return;
      setPermissionGranted(true);

      // Get initial position
      try {
        const pos = await Location.getCurrentPositionAsync({
          accuracy: Location.Accuracy.High,
        });
        if (!cancelled) {
          setLocation({
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
            accuracy: pos.coords.accuracy ?? 0,
          });
        }
      } catch {
        // will be updated by watch
      }

      // Watch position
      watchRef.current = await Location.watchPositionAsync(
        {
          accuracy: Location.Accuracy.High,
          timeInterval: trackInterval,
          distanceInterval: 10,
        },
        (pos) => {
          if (!cancelled) {
            setLocation({
              lat: pos.coords.latitude,
              lng: pos.coords.longitude,
              accuracy: pos.coords.accuracy ?? 0,
            });
          }
        },
      );
    })();

    return () => {
      cancelled = true;
      watchRef.current?.remove();
    };
  }, [trackInterval]);

  return { location, error, permissionGranted };
}

/**
 * Haversine distance between two points in meters.
 */
export function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const R = 6371000; // Earth radius in meters
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function toRad(deg: number) {
  return (deg * Math.PI) / 180;
}
