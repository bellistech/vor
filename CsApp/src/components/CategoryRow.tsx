import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { colors, typography, spacing, radius } from '../theme';
import type { CategorySummary } from '../core/types';

interface Props {
  category: CategorySummary;
  onPress: () => void;
}

export function CategoryRow({ category, onPress }: Props) {
  return (
    <TouchableOpacity style={styles.card} onPress={onPress} activeOpacity={0.7}>
      <Text style={styles.name}>{category.name}</Text>
      <Text style={styles.count}>{category.count} sheets</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    padding: spacing.lg,
    marginBottom: spacing.sm,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  name: {
    ...typography.body,
    color: colors.textPrimary,
  },
  count: {
    ...typography.label,
    color: colors.accentDim,
  },
});
