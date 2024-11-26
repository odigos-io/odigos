import type { InstrumentationRuleSpec } from '@/types';

const buildCard = (rule: InstrumentationRuleSpec) => {
  const { type, ruleName, notes, disabled, payloadCollection } = rule;

  const arr = [
    { title: 'Type', value: type || 'N/A' },
    { title: 'Status', value: String(!disabled) },
    { title: 'Name', value: ruleName || 'N/A' },
    { title: 'Notes', value: notes || 'N/A' },
  ];

  if (payloadCollection) {
    const str = Object.entries(payloadCollection)
      .filter(([key, val]) => !!val)
      .map(([key, val]) => key)
      .join(', ');

    arr.push({ title: 'Collect', value: str });
  }

  return arr;
};

export default buildCard;
