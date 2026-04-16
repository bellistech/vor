import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { colors, typography, spacing, radius } from '../theme';
import type { TopicSummary } from '../core/types';

interface Props {
  topic: TopicSummary;
  onPress: () => void;
  starred?: boolean;
  onStarPress?: () => void;
}

export function TopicRow({ topic, onPress, starred, onStarPress }: Props) {
  return (
    <TouchableOpacity style={styles.row} onPress={onPress} activeOpacity={0.7}>
      <View style={styles.content}>
        <Text style={styles.title} numberOfLines={1}>
          {topic.title || topic.name}
        </Text>
        <Text style={styles.category}>{topic.category}</Text>
      </View>
      {onStarPress !== undefined && (
        <TouchableOpacity
          onPress={onStarPress}
          style={styles.star}
          hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}>
          <Text style={[styles.starIcon, starred && styles.starIconActive]}>
            {starred ? '★' : '☆'}
          </Text>
        </TouchableOpacity>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  row: {
    backgroundColor: colors.bgCard,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    flexDirection: 'row',
    alignItems: 'center',
  },
  content: {
    flex: 1,
    marginRight: spacing.sm,
  },
  title: {
    ...typography.body,
    color: colors.textPrimary,
    marginBottom: 2,
  },
  category: {
    ...typography.meta,
    color: colors.textSecondary,
  },
  star: {
    padding: spacing.xs,
  },
  starIcon: {
    ...typography.subheading,
    color: colors.textSecondary,
  },
  starIconActive: {
    color: colors.accent,
  },
});
