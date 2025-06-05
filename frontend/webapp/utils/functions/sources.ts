import { getWorkloadId } from '@odigos/ui-kit/functions';
import { EntityTypes, type WorkloadId, type Source } from '@odigos/ui-kit/types';
import type { NamespaceSelectionFormData, SourceSelectionFormData } from '@odigos/ui-kit/store';
import type { NamespaceInstrumentInput, SourceConditions, SourceInstrumentInput } from '@/types';

export const addConditionToSources = ({ namespace, name, kind, conditions }: SourceConditions, sources: Source[]): Source | null => {
  const foundIdx = sources.findIndex((x) => x.namespace === namespace && x.name === name && x.kind === kind);
  if (foundIdx === -1) return null;

  if (sources[foundIdx].conditions) {
    return {
      ...sources[foundIdx],
      conditions: sources[foundIdx].conditions.concat(conditions),
    };
  } else {
    return {
      ...sources[foundIdx],
      conditions,
    };
  }
};

export const prepareSourcePayloads = (
  selectAppsList: SourceSelectionFormData,
  existingSources: Source[],
  selectedStreamName: string,
  handleInstrumentationCount: (toAddCount: number, toDeleteCount: number) => void,
  removeEntities: (entityType: EntityTypes, entityIds: WorkloadId[]) => void,
  addEntities: (entityType: EntityTypes, entities: Source[]) => void,
) => {
  let isEmpty = true;
  const payloads: SourceInstrumentInput[] = [];

  for (const [ns, items] of Object.entries(selectAppsList)) {
    if (items.length) {
      isEmpty = false;

      const mappedItems = items.map(({ name, kind, selected, currentStreamName }) => ({
        name,
        kind,
        // this is to map selected=undefined to selected=false
        selected: selected === undefined ? false : selected,

        // currentStreamName comes from the UI Kit, if it's missing we use selectedStreamName is a fallback,
        // we could rely on only the selectedStreamName, but if we want to override the selected then we need to use the currentStreamName
        // (for example - if we want to have a single page to manage all groups, then we need to override the selected)

        // TODO: uncomment when Data Streams are ready to use
        currentStreamName: '', // currentStreamName || selectedStreamName,
      }));
      const toDeleteFromStore: WorkloadId[] = [];
      const toUpdateInStore: Source[] = [];

      let toDeleteCount = 0;
      let toAddCount = 0;

      for (const item of mappedItems) {
        const foundExisting = existingSources.find((src) => src.namespace === ns && src.name === item.name && src.kind === item.kind);

        // Check if the instrumenting-source does not exist, this confirms an expected creation of the CRD
        if (item.selected && !foundExisting) {
          toAddCount++;
        }
        // Else the instrumenting-source should be updated in store to include the selected stream name
        else if (item.selected && foundExisting) {
          toUpdateInStore.push({ ...foundExisting, dataStreamNames: foundExisting.dataStreamNames.concat([selectedStreamName]) });
        }

        // Check if the uninstrumenting-source has 1 or none data streams, this confirms an expected deletion of the CRD
        else if (!item.selected && foundExisting && foundExisting.dataStreamNames.length <= 1) {
          toDeleteCount++;
          toDeleteFromStore.push(getWorkloadId(foundExisting));
        }
        // Else the uninstrumenting-source should be updated in store to exclude the selected stream name
        else if (!item.selected && foundExisting) {
          toUpdateInStore.push({ ...foundExisting, dataStreamNames: foundExisting.dataStreamNames.filter((name) => name !== selectedStreamName) });
        }
      }

      handleInstrumentationCount(toAddCount, toDeleteCount);
      removeEntities(EntityTypes.Source, toDeleteFromStore);
      addEntities(EntityTypes.Source, toUpdateInStore);

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
