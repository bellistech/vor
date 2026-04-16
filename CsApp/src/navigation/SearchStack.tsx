import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { SearchScreen } from '../screens/SearchScreen';
import { SheetScreen } from '../screens/SheetScreen';
import { DetailScreen } from '../screens/DetailScreen';
import { colors, typography } from '../theme';
import type { SearchStackParams } from './types';

const Stack = createNativeStackNavigator<SearchStackParams>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.bgSecondary },
  headerTintColor: colors.textPrimary,
  headerTitleStyle: typography.subheading,
  headerBackTitle: '',
};

export function SearchStack() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="Search"
        component={SearchScreen}
        options={{ title: 'Search' }}
      />
      <Stack.Screen name="Sheet" component={SheetScreen} />
      <Stack.Screen name="Detail" component={DetailScreen} />
    </Stack.Navigator>
  );
}
