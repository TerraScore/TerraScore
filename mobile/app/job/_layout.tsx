import { Stack } from 'expo-router';

export default function JobLayout() {
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="[id]/details" />
      <Stack.Screen name="[id]/navigate" />
      <Stack.Screen name="[id]/survey" />
    </Stack>
  );
}
