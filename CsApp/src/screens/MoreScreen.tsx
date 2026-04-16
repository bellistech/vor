import React from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
} from 'react-native';
import { useStats } from '../hooks/useCscore';
import { cscore } from '../core/cscore';
import { colors, typography, spacing, radius } from '../theme';

export function MoreScreen({ navigation }: any) {
  const stats = useStats();

  const handleRandom = async () => {
    try {
      const sheet = await cscore.randomTopic();
      navigation.navigate('Sheet', { topic: sheet.name, title: sheet.title });
    } catch {}
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {stats && (
        <View style={styles.statsGrid}>
          <StatTile value={stats.total_sheets} label="Sheets" />
          <StatTile value={stats.detail_pages} label="Details" />
          <StatTile value={stats.categories} label="Categories" />
          <StatTile value={stats.bookmarks} label="Starred" />
        </View>
      )}

      <View style={styles.divider} />

      <TouchableOpacity style={styles.randomButton} onPress={handleRandom}>
        <Text style={styles.randomButtonText}>Random Sheet</Text>
      </TouchableOpacity>

      <View style={styles.divider} />

      <Text style={styles.aboutTitle}>cs — Cheatsheet CLI</Text>
      <Text style={styles.aboutText}>
        685 cheatsheets across 59 categories
      </Text>
      <Text style={styles.aboutText}>
        CCNP DC/EI, CCIE EI/SP/Sec/Auto, JNCIE-SP/SEC, Linux+, CISSP, C|RAGE
      </Text>
      <Text style={styles.credit}>bellis.tech</Text>
    </ScrollView>
  );
}

function StatTile({ value, label }: { value: number; label: string }) {
  return (
    <View style={styles.statTile}>
      <Text style={styles.statValue}>{value}</Text>
      <Text style={styles.statLabel}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
  content: {
    padding: spacing.lg,
    paddingBottom: 40,
  },
  statsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.sm,
  },
  statTile: {
    flex: 1,
    minWidth: '45%',
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    padding: spacing.lg,
    alignItems: 'center',
  },
  statValue: {
    ...typography.heading,
    color: colors.accent,
    marginBottom: 4,
  },
  statLabel: {
    ...typography.label,
    color: colors.textSecondary,
  },
  divider: {
    height: 1,
    backgroundColor: colors.border,
    marginVertical: spacing.xl,
  },
  randomButton: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.accent,
    borderRadius: radius.md,
    padding: spacing.lg,
    alignItems: 'center',
  },
  randomButtonText: {
    ...typography.label,
    color: colors.accent,
  },
  aboutTitle: {
    ...typography.subheading,
    color: colors.textPrimary,
    marginBottom: spacing.sm,
  },
  aboutText: {
    ...typography.body,
    color: colors.textSecondary,
    marginBottom: spacing.xs,
  },
  credit: {
    ...typography.code,
    color: colors.accentDim,
    marginTop: spacing.lg,
  },
});
