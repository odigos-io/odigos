import { ActionsType } from '@/types'

export type ActionOption = {
  id: string
  type?: ActionsType
  label: string
  description?: string
  docsEndpoint?: string
  docsDescription?: string
  icon?: string
  items?: ActionOption[]
}

export const ACTION_OPTIONS: ActionOption[] = [
  {
    id: 'add_cluster_info',
    label: 'Add Cluster Info',
    description: 'Add static cluster-scoped attributes to your data.',
    type: ActionsType.ADD_CLUSTER_INFO,
    icon: '/icons/actions/addclusterinfo.svg',
    docsEndpoint: '/pipeline/actions/attributes/addclusterinfo',
    docsDescription:
      'The “Add Cluster Info” Odigos Action can be used to add resource attributes to telemetry signals originated from the k8s cluster where the Odigos is running.',
  },
  {
    id: 'delete_attribute',
    label: 'Delete Attribute',
    description: 'Delete attributes from logs, metrics, and traces.',
    type: ActionsType.DELETE_ATTRIBUTES,
    icon: '/icons/actions/deleteattribute.svg',
    docsEndpoint: '/pipeline/actions/attributes/deleteattribute',
    docsDescription: 'The “Delete Attribute” Odigos Action can be used to delete attributes from logs, metrics, and traces.',
  },
  {
    id: 'rename_attribute',
    label: 'Rename Attribute',
    description: 'Rename attributes in logs, metrics, and traces.',
    type: ActionsType.RENAME_ATTRIBUTES,
    icon: '/icons/actions/renameattribute.svg',
    docsEndpoint: '/pipeline/actions/attributes/rename-attribute',
    docsDescription:
      'The “Rename Attribute” Odigos Action can be used to rename attributes from logs, metrics, and traces. Different instrumentations might use different attribute names for similar information. This action let’s you to consolidate the names across your cluster.',
  },
  {
    id: 'pii-masking',
    label: 'PII Masking',
    description: 'Mask PII data in your traces.',
    type: ActionsType.PII_MASKING,
    icon: '/icons/actions/piimasking.svg',
    docsEndpoint: '/pipeline/actions/attributes/piimasking',
    docsDescription: 'The “PII Masking” Odigos Action can be used to mask PII data from span attribute values.',
  },
  {
    id: 'sampler',
    label: 'Samplers',
    icon: '/icons/actions/sampler.svg',
    items: [
      {
        id: 'error-sampler',
        label: 'Error Sampler',
        description: 'Sample errors based on percentage.',
        type: ActionsType.ERROR_SAMPLER,
        docsEndpoint: '/pipeline/actions/sampling/errorsampler',
        docsDescription: 'The “Error Sampler” Odigos Action is a Global Action that supports error sampling by filtering out non-error traces.',
      },
      {
        id: 'probabilistic-sampler',
        label: 'Probabilistic Sampler',
        description: 'Sample traces based on percentage.',
        type: ActionsType.PROBABILISTIC_SAMPLER,
        docsEndpoint: '/pipeline/actions/sampling/probabilisticsampler',
        docsDescription:
          'The “Probabilistic Sampler” Odigos Action supports probabilistic sampling based on a configured sampling percentage applied to the TraceID.',
      },
      {
        id: 'latency-action',
        label: 'Latency Action',
        description: 'Add latency to your traces.',
        type: ActionsType.LATENCY_SAMPLER,
        docsEndpoint: '/pipeline/actions/sampling/latencysampler',
        docsDescription:
          'The “Latency Sampler” Odigos Action is an Endpoint Action that samples traces based on their duration for a specific service and endpoint (HTTP route) filter.',
      },
    ],
  },
]
