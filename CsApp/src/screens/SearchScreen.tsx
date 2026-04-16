import React, { useCallback, useRef } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  StyleSheet,
} from 'react-native';
import { useSearch } from '../hooks/useCscore';
import { colors, typography, spacing, radius } from '../theme';
import type { SearchResult } from '../core/types';

export function SearchScreen({ navigation }: any) {
  const { results, search } = useSearch();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const onChangeText = useCallback(
    (text: string) => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
      debounceRef.current = setTimeout(() => search(text), 300);
    },
    [search],
  );

  const renderItem = useCallback(
    ({ item }: { item: SearchResult }) => (
      <TouchableOpacity
        style={styles.result}
        onPress={() =>
          navigation.navigate('Sheet', { topic: item.topic, title: item.topic })
        }
        activeOpacity={0.7}>
        <View style={styles.resultHeader}>
          <Text style={styles.topicName}>{item.topic}</Text>
          <Text style={styles.category}>{item.category}</Text>
        </View>
        {item.section && (
          <Text style={styles.section}>{item.section}</Text>
        )}
        {item.line && (
          <Text style={styles.line} numberOfLines={2}>
            {item.line}
          </Text>
        )}
      </TouchableOpacity>
    ),
    [navigation],
  );

  return (
    <View style={styles.container}>
      <View style={styles.inputWrapper}>
        <TextInput
          style={styles.input}
          placeholder="Search 685 sheets..."
          placeholderTextColor={colors.textSecondary}
          onChangeText={onChangeText}
          autoCorrect={false}
          autoCapitalize="none"
          clearButtonMode="while-editing"
          returnKeyType="search"
        />
      </View>
      {results && results.count === 0 ? (
        <View style={styles.empty}>
          <Text style={styles.emptyText}>No results</Text>
        </View>
      ) : (
        <FlatList
          data={results?.results ?? []}
          keyExtractor={(item, i) => `${item.topic}-${i}`}
          renderItem={renderItem}
          contentContainerStyle={styles.list}
          keyboardShouldPersistTaps="handled"
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
  inputWrapper: {
    padding: spacing.md,
    backgroundColor: colors.bgSecondary,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  input: {
    ...typography.body,
    color: colors.textPrimary,
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    height: 40,
  },
  list: {
    paddingBottom: spacing.xl,
  },
  result: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  resultHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  topicName: {
    ...typography.body,
    color: colors.textPrimary,
  },
  category: {
    ...typography.meta,
    color: colors.accentDim,
  },
  section: {
    ...typography.label,
    color: colors.accent,
    marginBottom: 2,
  },
  line: {
    ...typography.meta,
    color: colors.textSecondary,
  },
  empty: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emptyText: {
    ...typography.body,
    color: colors.textSecondary,
  },
});
