import { EntityTypes, type WorkloadId, type Source } from '@odigos/ui-kit/types';
import type { NamespaceSelectionFormData, SourceSelectionFormData } from '@odigos/ui-kit/store';
import type { InstrumentationInstancesHealth, NamespaceInstrumentInput, SourceInstrumentInput } from '@/types';

export const addConditionToSources = ({ namespace, name, kind, condition }: InstrumentationInstancesHealth, sources: Source[]): Source | null => {
  const foundIdx = sources.findIndex((x) => x.namespace === namespace && x.name === name && x.kind === kind);
  if (foundIdx === -1) return null;

  if (sources[foundIdx].conditions) {
    return {
      ...sources[foundIdx],
      conditions: sources[foundIdx].conditions.concat([condition]),
    };
  } else {
    return {
      ...sources[foundIdx],
      conditions: [condition],
    };
  }
};

export const prepareSourcePayloads = (
  selectAppsList: SourceSelectionFormData,
  handleInstrumentationCount: (toAddCount: number, toDeleteCount: number) => void,
  removeEntities: (entityType: EntityTypes, entityIds: WorkloadId[]) => void,
) => {
  let isEmpty = true;
  const payloads: SourceInstrumentInput[] = [];

  for (const [ns, items] of Object.entries(selectAppsList)) {
    if (items.length) {
      isEmpty = false;

      // this is to map selected=undefined to selected=false
      const mappedItems = items.map(({ name, kind, selected }) => ({ name, kind, selected: !selected ? false : true }));
      // this is to map delete-items from "SourceSelectionFormData" to "WorkloadId"
      const toDelete = mappedItems.filter((src) => !src.selected).map(({ name, kind }) => ({ namespace: ns, name, kind }));

      const toDeleteCount = toDelete.length;
      const toAddCount = mappedItems.length - toDeleteCount;

      handleInstrumentationCount(toAddCount, toDeleteCount);
      removeEntities(EntityTypes.Source, toDelete);

      payloads.push({ namespace: ns, sources: mappedItems });
    }
  }

  return { payloads, isEmpty };
};

export const prepareNamespacePayloads = (futureSelectAppsList: NamespaceSelectionFormData) => {
  let isEmpty = true;
  const payloads: NamespaceInstrumentInput[] = [];

  for (const [ns, futureSelected] of Object.entries(futureSelectAppsList)) {
    if (typeof futureSelected === 'boolean') {
      isEmpty = false;
      payloads.push({ name: ns, futureSelected });
    }
  }

  return { payloads, isEmpty };
};
