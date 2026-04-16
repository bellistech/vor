import React, { useLayoutEffect } from 'react';
import { View, FlatList, StyleSheet } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import type { BrowseStackParams } from '../navigation/types';
import { useCategoryTopics } from '../hooks/useCscore';
import { useBookmarks } from '../core/BookmarkContext';
import { TopicRow } from '../components/TopicRow';
import { LoadingView } from '../components/LoadingView';
import { colors, spacing } from '../theme';

type Props = NativeStackScreenProps<BrowseStackParams, 'TopicList'>;

export function TopicListScreen({ route, navigation }: Props) {
  const { category } = route.params;
  const { topics, loading } = useCategoryTopics(category);
  const { isStarred, toggle } = useBookmarks();

  useLayoutEffect(() => {
    navigation.setOptions({ title: category });
  }, [navigation, category]);

  if (loading) {
    return <LoadingView />;
  }

  return (
    <View style={styles.container}>
      <FlatList
        data={topics}
        keyExtractor={item => item.name}
        renderItem={({ item }) => (
          <TopicRow
            topic={item}
            starred={isStarred(item.name)}
            onPress={() =>
              navigation.navigate('Sheet', { topic: item.name, title: item.title || item.name })
            }
            onStarPress={() => toggle(item.name)}
          />
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
    paddingBottom: spacing.xl,
  },
});
