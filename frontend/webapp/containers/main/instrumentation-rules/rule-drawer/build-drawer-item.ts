import { PayloadCollectionType, type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';
import { deriveTypeFromRule } from '@/utils';

const buildDrawerItem = (id: string, item: InstrumentationRuleInput): InstrumentationRuleSpec => {
  const { ruleName, notes, disabled, workloads, instrumentationLibraries, payloadCollection } = item;

  return {
    ruleId: id,
    ruleName,
    type: deriveTypeFromRule(item),
    notes,
    disabled,
    workloads: workloads || [],
    // TODO: map "instrumentationLibraries" from params when this becomes relevant
    instrumentationLibraries: undefined,
    payloadCollection: {
      [PayloadCollectionType.HTTP_REQUEST]: payloadCollection[PayloadCollectionType.HTTP_REQUEST] || undefined,
      [PayloadCollectionType.HTTP_RESPONSE]: payloadCollection[PayloadCollectionType.HTTP_RESPONSE] || undefined,
      [PayloadCollectionType.DB_QUERY]: payloadCollection[PayloadCollectionType.DB_QUERY] || undefined,
      [PayloadCollectionType.MESSAGING]: payloadCollection[PayloadCollectionType.MESSAGING] || undefined,
    },
  };
};

export default buildDrawerItem;
