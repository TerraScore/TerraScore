import { View, Text, StyleSheet } from 'react-native';

interface Props {
  isOnline: boolean;
}

export function OfflineBanner({ isOnline }: Props) {
  if (isOnline) return null;

  return (
    <View style={styles.banner}>
      <Text style={styles.text}>You're offline — changes will sync when reconnected</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  banner: {
    backgroundColor: '#fbbf24',
    paddingVertical: 8,
    paddingHorizontal: 16,
    alignItems: 'center',
  },
  text: {
    fontSize: 13,
    fontWeight: '500',
    color: '#78350f',
  },
});
