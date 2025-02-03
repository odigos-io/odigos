import { deriveTypeFromRule } from '@odigos/ui-components';
import { type InstrumentationRuleSpec, type InstrumentationRuleInput, PayloadCollectionType, CodeAttributesType } from '@/types';

const buildDrawerItem = (id: string, formData: InstrumentationRuleInput, drawerItem: InstrumentationRuleSpec): InstrumentationRuleSpec => {
  const { ruleName, notes, disabled, payloadCollection, codeAttributes } = formData;
  const { mutable, profileName, workloads, instrumentationLibraries } = drawerItem;

  return {
    ruleId: id,
    ruleName,
    type: deriveTypeFromRule(formData),
    notes,
    disabled,
    mutable,
    profileName,
    workloads,
    instrumentationLibraries,
    payloadCollection: {
      [PayloadCollectionType.HTTP_REQUEST]: payloadCollection?.[PayloadCollectionType.HTTP_REQUEST] || undefined,
      [PayloadCollectionType.HTTP_RESPONSE]: payloadCollection?.[PayloadCollectionType.HTTP_RESPONSE] || undefined,
      [PayloadCollectionType.DB_QUERY]: payloadCollection?.[PayloadCollectionType.DB_QUERY] || undefined,
      [PayloadCollectionType.MESSAGING]: payloadCollection?.[PayloadCollectionType.MESSAGING] || undefined,
    },
    codeAttributes: {
      [CodeAttributesType.COLUMN]: codeAttributes?.[CodeAttributesType.COLUMN] || undefined,
      [CodeAttributesType.FILE_PATH]: codeAttributes?.[CodeAttributesType.FILE_PATH] || undefined,
      [CodeAttributesType.FUNCTION]: codeAttributes?.[CodeAttributesType.FUNCTION] || undefined,
      [CodeAttributesType.LINE_NUMBER]: codeAttributes?.[CodeAttributesType.LINE_NUMBER] || undefined,
      [CodeAttributesType.NAMESPACE]: codeAttributes?.[CodeAttributesType.NAMESPACE] || undefined,
      [CodeAttributesType.STACKTRACE]: codeAttributes?.[CodeAttributesType.STACKTRACE] || undefined,
    },
  };
};

export default buildDrawerItem;
