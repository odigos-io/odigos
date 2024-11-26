import { deriveTypeFromRule } from '@/utils';
import { PayloadCollectionType, type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';

const buildDrawerItem = (id: string, formData: InstrumentationRuleInput): InstrumentationRuleSpec => {
  const { ruleName, notes, disabled, workloads, instrumentationLibraries, payloadCollection } = formData;

  return {
    ruleId: id,
    ruleName,
    type: deriveTypeFromRule(formData),
    notes,
    disabled,
    workloads: workloads || [],
    payloadCollection: {
      [PayloadCollectionType.HTTP_REQUEST]: payloadCollection[PayloadCollectionType.HTTP_REQUEST] || undefined,
      [PayloadCollectionType.HTTP_RESPONSE]: payloadCollection[PayloadCollectionType.HTTP_RESPONSE] || undefined,
      [PayloadCollectionType.DB_QUERY]: payloadCollection[PayloadCollectionType.DB_QUERY] || undefined,
      [PayloadCollectionType.MESSAGING]: payloadCollection[PayloadCollectionType.MESSAGING] || undefined,
    },

    // TODO: map "instrumentationLibraries" (maybe ??)
    instrumentationLibraries: undefined,
  };
};

export default buildDrawerItem;
