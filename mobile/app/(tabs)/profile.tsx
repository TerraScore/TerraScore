import { View, Text, StyleSheet, TouchableOpacity, Switch, Alert } from 'react-native';
import { useAuthStore } from '@/stores/authStore';
import { api } from '@/services/api';
import { useState } from 'react';

export default function ProfileScreen() {
  const { agent, logout, fetchProfile } = useAuthStore();
  const [toggling, setToggling] = useState(false);

  const handleToggleAvailability = async (value: boolean) => {
    setToggling(true);
    try {
      await api.put('/v1/agents/me/availability', { is_online: value });
      await fetchProfile();
    } catch {
      Alert.alert('Error', 'Failed to update availability.');
    } finally {
      setToggling(false);
    }
  };

  const handleLogout = () => {
    Alert.alert('Logout', 'Are you sure you want to logout?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Logout', style: 'destructive', onPress: logout },
    ]);
  };

  return (
    <View style={styles.container}>
      <View style={styles.card}>
        <View style={styles.avatar}>
          <Text style={styles.avatarText}>
            {agent?.full_name?.charAt(0)?.toUpperCase() ?? '?'}
          </Text>
        </View>
        <Text style={styles.name}>{agent?.full_name ?? 'Agent'}</Text>
        <Text style={styles.phone}>{agent?.phone ?? ''}</Text>
        {agent?.tier && <Text style={styles.tier}>Tier: {agent.tier}</Text>}
      </View>

      <View style={styles.card}>
        <View style={styles.row}>
          <Text style={styles.rowLabel}>Available for jobs</Text>
          <Switch
            value={agent?.is_online ?? false}
            onValueChange={handleToggleAvailability}
            disabled={toggling}
            trackColor={{ false: '#ddd', true: '#93c5fd' }}
            thumbColor={agent?.is_online ? '#2563eb' : '#f4f3f4'}
          />
        </View>
      </View>

      {agent?.vehicle_type && (
        <View style={styles.card}>
          <InfoRow label="Vehicle" value={agent.vehicle_type} />
          {agent.preferred_radius_km && (
            <InfoRow label="Preferred Radius" value={`${agent.preferred_radius_km} km`} />
          )}
          {agent.total_jobs_completed != null && (
            <InfoRow label="Jobs Completed" value={String(agent.total_jobs_completed)} />
          )}
        </View>
      )}

      <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
        <Text style={styles.logoutText}>Logout</Text>
      </TouchableOpacity>
    </View>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={styles.rowValue}>{value}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
    padding: 16,
  },
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 20,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 2,
  },
  avatar: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: '#2563eb',
    alignItems: 'center',
    justifyContent: 'center',
    alignSelf: 'center',
    marginBottom: 12,
  },
  avatarText: {
    fontSize: 28,
    fontWeight: '700',
    color: '#fff',
  },
  name: {
    fontSize: 20,
    fontWeight: '600',
    color: '#1a1a2e',
    textAlign: 'center',
  },
  phone: {
    fontSize: 14,
    color: '#666',
    textAlign: 'center',
    marginTop: 4,
  },
  tier: {
    fontSize: 13,
    color: '#2563eb',
    textAlign: 'center',
    marginTop: 4,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 4,
  },
  rowLabel: {
    fontSize: 15,
    color: '#333',
  },
  rowValue: {
    fontSize: 15,
    color: '#666',
  },
  logoutButton: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    marginTop: 8,
    borderWidth: 1,
    borderColor: '#ef4444',
  },
  logoutText: {
    color: '#ef4444',
    fontSize: 16,
    fontWeight: '600',
  },
});
