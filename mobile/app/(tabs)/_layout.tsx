import { View } from 'react-native';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { Tabs } from 'expo-router';
import { useOfflineSync } from '@/hooks/useOfflineSync';
import { OfflineBanner } from '@/components/common/OfflineBanner';

function TabBarIcon(props: { name: React.ComponentProps<typeof FontAwesome>['name']; color: string }) {
  return <FontAwesome size={24} style={{ marginBottom: -3 }} {...props} />;
}

export default function TabLayout() {
  const { isOnline } = useOfflineSync();

  return (
    <View style={{ flex: 1 }}>
      <OfflineBanner isOnline={isOnline} />
      <Tabs
        screenOptions={{
          tabBarActiveTintColor: '#2563eb',
          tabBarInactiveTintColor: '#999',
          headerStyle: { backgroundColor: '#fff' },
          headerTitleStyle: { fontWeight: '600' },
        }}
      >
        <Tabs.Screen
          name="home"
          options={{
            title: 'Offers',
            tabBarIcon: ({ color }) => <TabBarIcon name="bolt" color={color} />,
          }}
        />
        <Tabs.Screen
          name="active"
          options={{
            title: 'Active Jobs',
            tabBarIcon: ({ color }) => <TabBarIcon name="briefcase" color={color} />,
          }}
        />
        <Tabs.Screen
          name="earnings"
          options={{
            title: 'Earnings',
            tabBarIcon: ({ color }) => <TabBarIcon name="inr" color={color} />,
          }}
        />
        <Tabs.Screen
          name="profile"
          options={{
            title: 'Profile',
            tabBarIcon: ({ color }) => <TabBarIcon name="user" color={color} />,
          }}
        />
      </Tabs>
    </View>
  );
}
