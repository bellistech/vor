import React, { useLayoutEffect } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
} from 'react-native';
import { useSheet } from '../hooks/useCscore';
import { useBookmarks } from '../core/BookmarkContext';
import { SheetViewer } from '../components/SheetViewer';
import { LoadingView } from '../components/LoadingView';
import { colors, typography, spacing, radius } from '../theme';

export function SheetScreen({ route, navigation }: any) {
  const { topic, title } = route.params;
  const { sheet, loading } = useSheet(topic);
  const { isStarred, toggle } = useBookmarks();
  const starred = isStarred(topic);

  useLayoutEffect(() => {
    navigation.setOptions({
      title,
      headerRight: () => (
        <TouchableOpacity
          onPress={() => toggle(topic)}
          hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}>
          <Text style={[styles.starIcon, starred && styles.starIconActive]}>
            {starred ? '★' : '☆'}
          </Text>
        </TouchableOpacity>
      ),
    });
  }, [navigation, title, topic, toggle, starred]);

  if (loading || !sheet) {
    return <LoadingView />;
  }

  const hasSeeAlso = sheet.see_also && sheet.see_also.length > 0;
  const hasFooter = sheet.has_detail || hasSeeAlso;

  return (
    <View style={styles.container}>
      {sheet.description ? (
        <Text style={styles.description} numberOfLines={3}>
          {sheet.description}
        </Text>
      ) : null}
      <SheetViewer content={sheet.content} />
      {hasFooter && (
        <View style={styles.footer}>
          {sheet.has_detail && (
            <TouchableOpacity
              style={styles.deepDiveButton}
              onPress={() =>
                navigation.push('Detail', {
                  topic,
                  title: `${sheet.title} — Deep Dive`,
                })
              }>
              <Text style={styles.deepDiveText}>Deep Dive →</Text>
            </TouchableOpacity>
          )}
          {hasSeeAlso && (
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              style={styles.seeAlsoScroll}
              contentContainerStyle={styles.seeAlsoContent}>
              {sheet.see_also!.map(related => (
                <TouchableOpacity
                  key={related}
                  style={styles.pill}
                  onPress={() =>
                    navigation.push('Sheet', { topic: related, title: related })
                  }>
                  <Text style={styles.pillText}>{related}</Text>
                </TouchableOpacity>
              ))}
            </ScrollView>
          )}
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
  description: {
    ...typography.meta,
    color: colors.textSecondary,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    backgroundColor: colors.bgSecondary,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  starIcon: {
    ...typography.subheading,
    color: colors.textSecondary,
  },
  starIconActive: {
    color: colors.accent,
  },
  footer: {
    backgroundColor: colors.bgSecondary,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    paddingVertical: spacing.sm,
  },
  deepDiveButton: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
  },
  deepDiveText: {
    ...typography.label,
    color: colors.accent,
  },
  seeAlsoScroll: {
    flexGrow: 0,
  },
  seeAlsoContent: {
    paddingHorizontal: spacing.md,
    gap: spacing.sm,
  },
  pill: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.full,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
  },
  pillText: {
    ...typography.meta,
    color: colors.textSecondary,
  },
});
