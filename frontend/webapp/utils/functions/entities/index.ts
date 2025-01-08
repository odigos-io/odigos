import { InstrumentationRuleSpec, K8sActualSource, ActionDataParsed, ActualDestination } from '@/types';

type Item = InstrumentationRuleSpec | K8sActualSource | ActionDataParsed | ActualDestination;

export const getEntityItemId = (item: Item): string | { kind: string; name: string; namespace: string } | undefined => {
  if ('ruleId' in item) {
    // InstrumentationRuleSpec
    return item.ruleId;
  } else if ('id' in item) {
    // ActualDestination or ActionDataParsed
    return item.id;
  } else if ('kind' in item && 'name' in item && 'namespace' in item) {
    // K8sActualSource
    return {
      kind: item.kind,
      name: item.name,
      namespace: item.namespace,
    };
  }

  // If the type doesn't match any of the known ones, return undefined
  console.error('Unhandled item type', item);
  return undefined;
};
