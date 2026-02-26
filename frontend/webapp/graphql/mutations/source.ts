import { gql } from '@apollo/client';

export const PERSIST_SOURCES = gql`
  mutation PersistSources($sources: [PersistNamespaceSourceInput!]!) {
    persistK8sSources(sources: $sources)
  }
`;

export const UPDATE_K8S_ACTUAL_SOURCE = gql`
  mutation UpdateK8sActualSource($sourceId: K8sSourceId!, $patchSourceRequest: PatchSourceRequestInput!) {
    updateK8sActualSource(sourceId: $sourceId, patchSourceRequest: $patchSourceRequest)
  }
`;

export const RESTART_WORKLOADS = gql`
  mutation RestartWorkloads($sourceIds: [K8sSourceId!]!) {
    restartWorkloads(sourceIds: $sourceIds)
  }
`;

export const RESTART_POD = gql`
  mutation RestartPod($namespace: String!, $name: String!) {
    restartPod(namespace: $namespace, name: $name)
  }
`;

export const RECOVER_FROM_ROLLBACK = gql`
  mutation RecoverFromRollbackForWorkload($sourceId: K8sSourceId!) {
    recoverFromRollbackForWorkload(sourceId: $sourceId)
  }
`;
