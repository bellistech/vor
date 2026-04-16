import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { StarredScreen } from '../screens/StarredScreen';
import { SheetScreen } from '../screens/SheetScreen';
import { DetailScreen } from '../screens/DetailScreen';
import { colors, typography } from '../theme';
import type { StarredStackParams } from './types';

const Stack = createNativeStackNavigator<StarredStackParams>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.bgSecondary },
  headerTintColor: colors.textPrimary,
  headerTitleStyle: typography.subheading,
  headerBackTitle: '',
};

export function StarredStack() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="Starred"
        component={StarredScreen}
        options={{ title: 'Starred' }}
      />
      <Stack.Screen name="Sheet" component={SheetScreen} />
      <Stack.Screen name="Detail" component={DetailScreen} />
    </Stack.Navigator>
  );
}
