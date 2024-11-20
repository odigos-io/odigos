import type { InstrumentationRuleSpec } from '@/types';

const buildCardFromRuleSpec = (rule: InstrumentationRuleSpec) => {
  const { type, ruleName, notes, disabled, payloadCollection } = rule as InstrumentationRuleSpec;

  const arr = [
    { title: 'Type', value: type || 'N/A' },
    { title: 'Status', value: String(!disabled) },
    { title: 'Name', value: ruleName || 'N/A' },
    { title: 'Notes', value: notes || 'N/A' },
  ];

  if (payloadCollection) {
    let str = '';
    const entries = Object.entries(payloadCollection).filter(([key, val]) => !!val);
    entries.forEach(([key, val], idx) => {
      str += key;
      if (idx < entries.length - 1) str += ', ';
    });

    arr.push({ title: 'Collect', value: str });
  }

  return arr;
};

export default buildCardFromRuleSpec;
