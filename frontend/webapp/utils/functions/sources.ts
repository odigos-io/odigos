import { getWorkloadId } from '@odigos/ui-kit/functions';
import { EntityTypes, type WorkloadId, type Source, type Workload } from '@odigos/ui-kit/types';
import type { NamespaceSelectionFormData, SourceSelectionFormData } from '@odigos/ui-kit/store';
import type { NamespaceInstrumentInput, SourceConditions, SourceInstrumentInput } from '@/types';

export const addConditionToSources = ({ namespace, name, kind, conditions }: SourceConditions, sources: Source[]): Source | null => {
  const foundIdx = sources.findIndex((x) => x.namespace === namespace && x.name === name && x.kind === kind);
  if (foundIdx === -1) return null;

  if (sources[foundIdx].conditions) {
    return {
      ...sources[foundIdx],
      conditions: (sources[foundIdx].conditions ?? []).concat(conditions),
    };
  }

  return {
    ...sources[foundIdx],
    conditions,
  };
};

export const addAgentInjectionStatusToSources = ({ id: { namespace, name, kind }, podsAgentInjectionStatus }: Workload, sources: Source[]): Source | null => {
  const foundIdx = sources.findIndex((x) => x.namespace === namespace && x.name === name && x.kind === kind);
  if (foundIdx === -1) return null;

  return {
    ...sources[foundIdx],
    podsAgentInjectionStatus,
  };
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
  const payload: SourceInstrumentInput = { sources: [] };

  for (const [ns, items] of Object.entries(selectAppsList)) {
    if (items.length) {
      isEmpty = false;

      const mappedItems = items.map(({ namespace, name, kind, selected, currentStreamName }) => ({
        namespace,
        name,
        kind,
        // this is to map selected=undefined to selected=false
        selected: selected === undefined ? false : selected,

        // currentStreamName comes from the UI Kit, if it's missing we use selectedStreamName as a fallback,
        // we could rely on only the selectedStreamName, but if we want to override the selected then we need to use the currentStreamName
        // (for example - if we want to have a single page to manage all groups, then we need to override the selected)
        currentStreamName: currentStreamName || selectedStreamName,
      }));

      const toAddToStore: Source[] = [];
      const toUpdateInStore: Source[] = [];
      const toDeleteFromStore: WorkloadId[] = [];

      let toAddCount = 0;
      let toDeleteCount = 0;

      for (const item of mappedItems) {
        const foundExisting = existingSources.find((src) => src.namespace === ns && src.name === item.name && src.kind === item.kind);

        // Check if the instrumenting-source does not exist, this confirms an expected creation of the CRD
        if (item.selected && !foundExisting) {
          toAddCount++;
          toAddToStore.push({ ...getWorkloadId(item), dataStreamNames: [selectedStreamName] });
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
      addEntities(EntityTypes.Source, toAddToStore);
      addEntities(EntityTypes.Source, toUpdateInStore);
      removeEntities(EntityTypes.Source, toDeleteFromStore);

      payload.sources.push(...mappedItems);
    }
  }

  return { payload, isEmpty };
};

export const prepareNamespacePayloads = (futureSelectAppsList: NamespaceSelectionFormData, selectedStreamName: string): { payload: NamespaceInstrumentInput; isEmpty: boolean } => {
  let isEmpty = true;
  const payload: NamespaceInstrumentInput = { namespaces: [] };

  for (const [ns, { selected, currentStreamName }] of Object.entries(futureSelectAppsList)) {
    if (typeof selected === 'boolean') {
      isEmpty = false;
      payload.namespaces.push({
        namespace: ns,
        selected: selected,

        // currentStreamName comes from the UI Kit, if it's missing we use selectedStreamName as a fallback,
        // we could rely on only the selectedStreamName, but if we want to override the selected then we need to use the currentStreamName
        // (for example - if we want to have a single page to manage all groups, then we need to override the selected)
        currentStreamName: currentStreamName || selectedStreamName,
      });
    }
  }

  return { payload, isEmpty };
};
