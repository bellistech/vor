import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
} from 'react-native';
import { useCalc, useSubnet } from '../hooks/useCscore';
import { colors, typography, spacing, radius } from '../theme';
import type { SubnetResponse } from '../core/types';

export function ToolsScreen() {
  const { result: calcResult, error: calcError, evaluate } = useCalc();
  const { result: subnetResult, error: subnetError, calculate } = useSubnet();
  const [calcExpr, setCalcExpr] = useState('');
  const [subnetInput, setSubnetInput] = useState('');

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.content}
      keyboardShouldPersistTaps="handled">
      <Text style={styles.sectionHeader}>Calculator</Text>
      <Text style={styles.sectionHint}>
        e.g. 1 GB / 8 Mbps  •  0xFF & 0xF0  •  sqrt(2)
      </Text>
      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          placeholder="1 GB / 8 Mbps"
          placeholderTextColor={colors.textSecondary}
          value={calcExpr}
          onChangeText={setCalcExpr}
          onSubmitEditing={() => evaluate(calcExpr)}
          autoCorrect={false}
          autoCapitalize="none"
          returnKeyType="done"
        />
        <TouchableOpacity
          style={styles.evalButton}
          onPress={() => evaluate(calcExpr)}>
          <Text style={styles.evalButtonText}>=</Text>
        </TouchableOpacity>
      </View>
      {calcError ? (
        <Text style={styles.errorText}>{calcError}</Text>
      ) : calcResult ? (
        <View style={styles.resultBox}>
          <Text style={styles.resultMain}>{calcResult.formatted}</Text>
          {calcResult.unit ? (
            <Text style={styles.resultDetail}>unit: {calcResult.unit}</Text>
          ) : null}
          {calcResult.hex ? (
            <Text style={styles.resultDetail}>hex: {calcResult.hex}</Text>
          ) : null}
          {calcResult.oct ? (
            <Text style={styles.resultDetail}>oct: {calcResult.oct}</Text>
          ) : null}
          {calcResult.bin ? (
            <Text style={styles.resultDetail}>bin: {calcResult.bin}</Text>
          ) : null}
        </View>
      ) : null}

      <View style={styles.divider} />

      <Text style={styles.sectionHeader}>Subnet Calculator</Text>
      <Text style={styles.sectionHint}>IPv4 or IPv6 CIDR notation</Text>
      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          placeholder="10.0.0.0/24"
          placeholderTextColor={colors.textSecondary}
          value={subnetInput}
          onChangeText={setSubnetInput}
          onSubmitEditing={() => calculate(subnetInput)}
          autoCorrect={false}
          autoCapitalize="none"
          keyboardType="ascii-capable"
          returnKeyType="done"
        />
        <TouchableOpacity
          style={styles.evalButton}
          onPress={() => calculate(subnetInput)}>
          <Text style={styles.evalButtonText}>=</Text>
        </TouchableOpacity>
      </View>
      {subnetError ? (
        <Text style={styles.errorText}>{subnetError}</Text>
      ) : subnetResult ? (
        <View style={styles.resultBox}>
          <SubnetRow label="Network" value={subnetResult.network} />
          {subnetResult.broadcast ? (
            <SubnetRow label="Broadcast" value={subnetResult.broadcast} />
          ) : null}
          {subnetResult.netmask ? (
            <SubnetRow label="Netmask" value={subnetResult.netmask} />
          ) : null}
          <SubnetRow label="First Host" value={subnetResult.first_host} />
          <SubnetRow label="Last Host" value={subnetResult.last_host} />
          <SubnetRow label="Total Hosts" value={subnetResult.total_hosts} />
          <SubnetRow label="Usable Hosts" value={subnetResult.usable_hosts} />
        </View>
      ) : null}
    </ScrollView>
  );
}

function SubnetRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.subnetRow}>
      <Text style={styles.subnetLabel}>{label}</Text>
      <Text style={styles.subnetValue}>{value}</Text>
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
  sectionHeader: {
    ...typography.subheading,
    color: colors.textPrimary,
    marginBottom: spacing.xs,
  },
  sectionHint: {
    ...typography.meta,
    color: colors.textSecondary,
    marginBottom: spacing.md,
  },
  inputRow: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginBottom: spacing.sm,
  },
  input: {
    flex: 1,
    ...typography.code,
    color: colors.textPrimary,
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    height: 44,
  },
  evalButton: {
    backgroundColor: colors.accent,
    borderRadius: radius.md,
    width: 44,
    height: 44,
    justifyContent: 'center',
    alignItems: 'center',
  },
  evalButtonText: {
    ...typography.subheading,
    color: '#ffffff',
  },
  errorText: {
    ...typography.meta,
    color: colors.error,
    marginBottom: spacing.md,
  },
  resultBox: {
    backgroundColor: colors.bgCard,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    padding: spacing.md,
    marginBottom: spacing.md,
  },
  resultMain: {
    ...typography.subheading,
    color: colors.accent,
    marginBottom: spacing.xs,
  },
  resultDetail: {
    ...typography.code,
    color: colors.textSecondary,
    marginTop: 2,
  },
  divider: {
    height: 1,
    backgroundColor: colors.border,
    marginVertical: spacing.xl,
  },
  subnetRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 4,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  subnetLabel: {
    ...typography.meta,
    color: colors.textSecondary,
  },
  subnetValue: {
    ...typography.code,
    color: colors.textPrimary,
  },
});
