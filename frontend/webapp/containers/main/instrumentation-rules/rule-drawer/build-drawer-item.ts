import { deriveTypeFromRule } from '@/utils';
import { type InstrumentationRuleSpec, type InstrumentationRuleInput, PayloadCollectionType } from '@/types';

const buildDrawerItem = (id: string, formData: InstrumentationRuleInput, drawerItem: InstrumentationRuleSpec): InstrumentationRuleSpec => {
  const { ruleName, notes, disabled, payloadCollection } = formData;
  const { mutable, profileName, workloads, instrumentationLibraries } = drawerItem;

  return {
    ruleId: id,
    ruleName,
    type: deriveTypeFromRule(formData),
    notes,
    disabled,
    mutable,
    profileName,
    payloadCollection: {
      [PayloadCollectionType.HTTP_REQUEST]: payloadCollection[PayloadCollectionType.HTTP_REQUEST] || undefined,
      [PayloadCollectionType.HTTP_RESPONSE]: payloadCollection[PayloadCollectionType.HTTP_RESPONSE] || undefined,
      [PayloadCollectionType.DB_QUERY]: payloadCollection[PayloadCollectionType.DB_QUERY] || undefined,
      [PayloadCollectionType.MESSAGING]: payloadCollection[PayloadCollectionType.MESSAGING] || undefined,
    },
    workloads,
    instrumentationLibraries,
  };
};

export default buildDrawerItem;
