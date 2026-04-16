import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { MoreScreen } from '../screens/MoreScreen';
import { SheetScreen } from '../screens/SheetScreen';
import { DetailScreen } from '../screens/DetailScreen';
import { colors, typography } from '../theme';
import type { MoreStackParams } from './types';

const Stack = createNativeStackNavigator<MoreStackParams>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.bgSecondary },
  headerTintColor: colors.textPrimary,
  headerTitleStyle: typography.subheading,
  headerBackTitle: '',
};

export function MoreStack() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="More"
        component={MoreScreen}
        options={{ title: 'More' }}
      />
      <Stack.Screen name="Sheet" component={SheetScreen} />
      <Stack.Screen name="Detail" component={DetailScreen} />
    </Stack.Navigator>
  );
}
