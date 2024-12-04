import { gql } from '@apollo/client';

export const PERSIST_SOURCE = gql`
  mutation PersistSources($namespace: String!, $sources: [PersistNamespaceSourceInput!]!) {
    persistK8sSources(namespace: $namespace, sources: $sources)
  }
`;

export const UPDATE_K8S_ACTUAL_SOURCE = gql`
  mutation UpdateK8sActualSource($sourceId: K8sSourceId!, $patchSourceRequest: PatchSourceRequestInput!) {
    updateK8sActualSource(sourceId: $sourceId, patchSourceRequest: $patchSourceRequest)
  }
`;
