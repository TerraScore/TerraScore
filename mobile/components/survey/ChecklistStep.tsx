import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import type { SurveyStep } from '@/types/api';

type Answer = 'yes' | 'no' | 'na';

interface Props {
  step: SurveyStep;
  value: Answer | null;
  onChange: (value: Answer) => void;
}

const OPTIONS: { key: Answer; label: string; color: string; bg: string }[] = [
  { key: 'yes', label: 'Yes', color: '#16a34a', bg: '#dcfce7' },
  { key: 'no', label: 'No', color: '#dc2626', bg: '#fef2f2' },
  { key: 'na', label: 'N/A', color: '#6b7280', bg: '#f3f4f6' },
];

export function ChecklistStep({ step, value, onChange }: Props) {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>{step.title}</Text>
      {step.description && <Text style={styles.description}>{step.description}</Text>}

      <View style={styles.options}>
        {OPTIONS.map((opt) => {
          const selected = value === opt.key;
          return (
            <TouchableOpacity
              key={opt.key}
              style={[
                styles.option,
                selected && { backgroundColor: opt.bg, borderColor: opt.color },
              ]}
              onPress={() => onChange(opt.key)}
            >
              <Text
                style={[
                  styles.optionText,
                  selected && { color: opt.color, fontWeight: '600' },
                ]}
              >
                {opt.label}
              </Text>
            </TouchableOpacity>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
  },
  title: {
    fontSize: 17,
    fontWeight: '600',
    color: '#1a1a2e',
    marginBottom: 6,
  },
  description: {
    fontSize: 14,
    color: '#666',
    marginBottom: 16,
  },
  options: {
    flexDirection: 'row',
    gap: 12,
  },
  option: {
    flex: 1,
    paddingVertical: 14,
    borderRadius: 10,
    borderWidth: 1.5,
    borderColor: '#e5e7eb',
    alignItems: 'center',
    backgroundColor: '#fff',
  },
  optionText: {
    fontSize: 15,
    color: '#666',
  },
});
