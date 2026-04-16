import React from 'react';
import { View, Text, ScrollView, TouchableOpacity, StyleSheet } from 'react-native';
import { useDetail } from '../hooks/useCscore';
import { SheetViewer } from '../components/SheetViewer';
import { LoadingView } from '../components/LoadingView';
import { colors, typography, spacing, radius } from '../theme';

export function DetailScreen({ route, navigation }: any) {
  const { topic } = route.params;
  const { detail, loading } = useDetail(topic);

  if (loading || !detail) {
    return <LoadingView />;
  }

  const hasPrereqs = detail.prerequisites && detail.prerequisites.length > 0;
  const hasMeta = detail.complexity || hasPrereqs;

  return (
    <View style={styles.container}>
      {hasMeta && (
        <View style={styles.meta}>
          {detail.complexity && (
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{detail.complexity}</Text>
            </View>
          )}
          {hasPrereqs && (
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.prereqRow}>
              {detail.prerequisites!.map(p => (
                <TouchableOpacity
                  key={p}
                  style={styles.prereqPill}
                  onPress={() => navigation.push('Sheet', { topic: p, title: p })}>
                  <Text style={styles.prereqText}>{p}</Text>
                </TouchableOpacity>
              ))}
            </ScrollView>
          )}
        </View>
      )}
      <SheetViewer content={detail.content} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
  meta: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    backgroundColor: colors.bgSecondary,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    gap: spacing.sm,
  },
  badge: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.accentDim,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
  },
  badgeText: {
    ...typography.meta,
    color: colors.accent,
    textTransform: 'uppercase',
  },
  prereqRow: {
    gap: spacing.xs,
  },
  prereqPill: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.full,
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
  },
  prereqText: {
    ...typography.meta,
    color: colors.textSecondary,
  },
});
