import type { InstrumentationRuleSpec } from '@/types';
import type { DataCardRow } from '@/reuseable-components';

const buildCard = (rule: InstrumentationRuleSpec) => {
  const { type, ruleName, notes, disabled, payloadCollection } = rule;

  const arr: DataCardRow[] = [
    { title: 'Type', value: type },
    { title: 'Name', value: ruleName },
    { title: 'Notes', value: notes },
    { type: 'divider' },
    { title: 'Status', type: 'active-status', value: String(!disabled) },
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
