import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Text, StyleSheet } from 'react-native';
import { BrowseStack } from './BrowseStack';
import { SearchStack } from './SearchStack';
import { ToolsStack } from './ToolsStack';
import { StarredStack } from './StarredStack';
import { MoreStack } from './MoreStack';
import { colors, typography } from '../theme';

const Tab = createBottomTabNavigator();

function TabIcon({ label, focused }: { label: string; focused: boolean }) {
  return (
    <Text style={[styles.tabIcon, focused && styles.tabIconActive]}>
      {label}
    </Text>
  );
}

export function TabNavigator() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarStyle: styles.tabBar,
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textSecondary,
        tabBarLabelStyle: styles.tabLabel,
      }}>
      <Tab.Screen
        name="Browse"
        component={BrowseStack}
        options={{
          tabBarIcon: ({ focused }) => <TabIcon label="[]" focused={focused} />,
        }}
      />
      <Tab.Screen
        name="Search"
        component={SearchStack}
        options={{
          tabBarIcon: ({ focused }) => <TabIcon label="/$" focused={focused} />,
        }}
      />
      <Tab.Screen
        name="Tools"
        component={ToolsStack}
        options={{
          tabBarIcon: ({ focused }) => <TabIcon label=">_" focused={focused} />,
        }}
      />
      <Tab.Screen
        name="Starred"
        component={StarredStack}
        options={{
          tabBarIcon: ({ focused }) => <TabIcon label="*" focused={focused} />,
        }}
      />
      <Tab.Screen
        name="More"
        component={MoreStack}
        options={{
          tabBarIcon: ({ focused }) => <TabIcon label="..." focused={focused} />,
        }}
      />
    </Tab.Navigator>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    backgroundColor: colors.bgSecondary,
    borderTopColor: colors.border,
    borderTopWidth: 1,
  },
  tabLabel: {
    ...typography.meta,
    fontSize: 10,
  },
  tabIcon: {
    ...typography.code,
    fontSize: 16,
    color: colors.textSecondary,
  },
  tabIconActive: {
    color: colors.accent,
  },
});
