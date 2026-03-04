import { getWorkloadId } from '@odigos/ui-kit/functions';
import {
  EntityTypes,
  DesiredStateProgress,
  StatusType,
  OtherStatus,
  type WorkloadId,
  type Source,
  type SourceContainer,
  type Condition,
  type DesiredConditionStatus,
  type ProgrammingLanguages,
  type OtelDistroName,
} from '@odigos/ui-kit/types';
import type { NamespaceSelectionFormData, SourceSelectionFormData } from '@odigos/ui-kit/store';
import type { NamespaceInstrumentInput, SourceInstrumentInput, WorkloadResponse, K8sWorkloadContainerResponse, K8sWorkloadConditions } from '@/types';

function mapDesiredStatusToConditionStatus(status: DesiredStateProgress): StatusType | OtherStatus {
  switch (status) {
    case DesiredStateProgress.Failure:
      return StatusType.Error;
    case DesiredStateProgress.Notice:
      return StatusType.Warning;
    case DesiredStateProgress.Pending:
    case DesiredStateProgress.Waiting:
      return OtherStatus.Loading;
    case DesiredStateProgress.Unsupported:
    case DesiredStateProgress.Disabled:
      return OtherStatus.Disabled;
    case DesiredStateProgress.Error:
    case DesiredStateProgress.Success:
    case DesiredStateProgress.Irrelevant:
    case DesiredStateProgress.Unknown:
    default:
      return StatusType.Default;
  }
}

function mapContainerToSourceContainer(c: K8sWorkloadContainerResponse): SourceContainer {
  return {
    containerName: c.containerName,
    language: (c.runtimeInfo?.language?.toLowerCase() ?? 'unknown') as ProgrammingLanguages,
    runtimeVersion: c.runtimeInfo?.runtimeVersion ?? '',
    overriden: c.overrides != null,
    instrumented: c.agentEnabled?.agentEnabled ?? false,
    instrumentationMessage: c.agentEnabled?.agentEnabledStatus?.message ?? '',
    otelDistroName: (c.agentEnabled?.otelDistroName as OtelDistroName) ?? null,
  };
}

function mapConditionsToConditionArray(conditions: K8sWorkloadConditions | null): Condition[] | null {
  if (!conditions) return null;

  const result: Condition[] = [];
  const fields: (keyof K8sWorkloadConditions)[] = ['runtimeDetection', 'agentInjectionEnabled', 'rollout', 'agentInjected', 'processesAgentHealth', 'expectingTelemetry'];

  for (const field of fields) {
    const dcs: DesiredConditionStatus | null = conditions[field];
    if (!dcs) continue;

    result.push({
      type: dcs.name ?? field,
      status: mapDesiredStatusToConditionStatus(dcs.status),
      reason: dcs.reasonEnum ?? null,
      message: dcs.message ?? null,
      lastTransitionTime: '',
    });
  }

  return result.length > 0 ? result : null;
}

export function mapWorkloadToSource(w: WorkloadResponse): Source {
  return {
    namespace: w.id.namespace,
    kind: w.id.kind,
    name: w.id.name,
    selected: w.markedForInstrumentation?.markedForInstrumentation ?? false,
    otelServiceName: w.serviceName ?? '',
    numberOfInstances: w.numberOfInstances ?? undefined,
    dataStreamNames: w.dataStreamNames,
    containers: w.containers ? w.containers.map(mapContainerToSourceContainer) : null,
    conditions: mapConditionsToConditionArray(w.conditions),
    detectedLanguages: w.runtimeInfo?.detectedLanguages?.map((lang) => lang.toLowerCase() as ProgrammingLanguages) ?? null,
    workloadOdigosHealthStatus: w.workloadOdigosHealthStatus ?? null,
    podsAgentInjectionStatus: w.podsAgentInjectionStatus,
    rollbackOccurred: w.rollbackOccurred,
  };
}

export function sortSources(sources: Source[]): Source[] {
  return [...sources].sort((a, b) => {
    const ns = a.namespace.localeCompare(b.namespace);
    if (ns !== 0) return ns;
    return a.name.localeCompare(b.name);
  });
}

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
