import { useEffect, useRef, useState } from 'react';
import NetInfo from '@react-native-community/netinfo';
import { AppState } from 'react-native';
import { runSync } from '@/services/sync';

export function useOfflineSync() {
  const [isOnline, setIsOnline] = useState(true);
  const wasOffline = useRef(false);

  useEffect(() => {
    // Listen for connectivity changes
    const unsubscribe = NetInfo.addEventListener((state) => {
      const online = state.isConnected === true && state.isInternetReachable !== false;
      setIsOnline(online);

      if (online && wasOffline.current) {
        // Just came back online — trigger sync
        wasOffline.current = false;
        runSync();
      }

      if (!online) {
        wasOffline.current = true;
      }
    });

    // Also sync when app comes to foreground
    const appStateSub = AppState.addEventListener('change', (nextState) => {
      if (nextState === 'active') {
        NetInfo.fetch().then((state) => {
          if (state.isConnected) {
            runSync();
          }
        });
      }
    });

    // Initial sync attempt
    runSync();

    return () => {
      unsubscribe();
      appStateSub.remove();
    };
  }, []);

  return { isOnline };
}
