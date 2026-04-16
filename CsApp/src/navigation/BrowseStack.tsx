import React from 'react';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { CategoriesScreen } from '../screens/CategoriesScreen';
import { TopicListScreen } from '../screens/TopicListScreen';
import { SheetScreen } from '../screens/SheetScreen';
import { DetailScreen } from '../screens/DetailScreen';
import { colors, typography } from '../theme';
import type { BrowseStackParams } from './types';

const Stack = createNativeStackNavigator<BrowseStackParams>();

const screenOptions = {
  headerStyle: { backgroundColor: colors.bgSecondary },
  headerTintColor: colors.textPrimary,
  headerTitleStyle: typography.subheading,
  headerBackTitle: '',
};

export function BrowseStack() {
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="Categories"
        component={CategoriesScreen}
        options={{ title: 'Browse' }}
      />
      <Stack.Screen name="TopicList" component={TopicListScreen} />
      <Stack.Screen name="Sheet" component={SheetScreen} />
      <Stack.Screen name="Detail" component={DetailScreen} />
    </Stack.Navigator>
  );
}
