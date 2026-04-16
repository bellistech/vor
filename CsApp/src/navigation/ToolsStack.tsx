import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ToolsScreen } from '../screens/ToolsScreen';
import { colors, typography } from '../theme';
import type { ToolsStackParams } from './types';

const Stack = createNativeStackNavigator<ToolsStackParams>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.bgSecondary },
  headerTintColor: colors.textPrimary,
  headerTitleStyle: typography.subheading,
};

export function ToolsStack() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="Tools"
        component={ToolsScreen}
        options={{ title: 'Tools' }}
      />
    </Stack.Navigator>
  );
}
