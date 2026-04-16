import React from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
} from 'react-native';
import { useBookmarks } from '../core/BookmarkContext';
import { colors, typography, spacing } from '../theme';

export function StarredScreen({ navigation }: any) {
  const { starred, toggle } = useBookmarks();
  const items = Array.from(starred);

  if (items.length === 0) {
    return (
      <View style={styles.empty}>
        <Text style={styles.emptyText}>No starred sheets yet.</Text>
        <Text style={styles.emptyHint}>Tap ★ on any sheet to bookmark it.</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <FlatList
        data={items}
        keyExtractor={item => item}
        renderItem={({ item }) => (
          <TouchableOpacity
            style={styles.row}
            onPress={() =>
              navigation.navigate('Sheet', { topic: item, title: item })
            }
            activeOpacity={0.7}>
            <Text style={styles.topicName}>{item}</Text>
            <TouchableOpacity
              onPress={() => toggle(item)}
              hitSlop={{ top: 10, bottom: 10, left: 10, right: 10 }}>
              <Text style={styles.star}>★</Text>
            </TouchableOpacity>
          </TouchableOpacity>
        )}
        contentContainerStyle={styles.list}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
  list: {
    paddingBottom: 32,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  topicName: {
    ...typography.body,
    color: colors.textPrimary,
    flex: 1,
    marginRight: spacing.sm,
  },
  star: {
    ...typography.subheading,
    color: colors.accent,
  },
  empty: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  emptyText: {
    ...typography.body,
    color: colors.textPrimary,
    marginBottom: 8,
    textAlign: 'center',
  },
  emptyHint: {
    ...typography.meta,
    color: colors.textSecondary,
    textAlign: 'center',
  },
});
