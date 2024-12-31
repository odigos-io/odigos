import { SVG } from '@/assets';
import { ActionsType } from '@/types';
import { getActionIcon, SignalUppercase } from '@/utils';

export type ActionOption = {
  id: string;
  type?: ActionsType;
  label: string;
  description?: string;
  docsEndpoint?: string;
  docsDescription?: string;
  icon?: SVG;
  items?: ActionOption[];
  allowedSignals?: SignalUppercase[];
};

export const ACTION_OPTIONS: ActionOption[] = [
  {
    id: 'attributes',
    label: 'Attributes',
    icon: getActionIcon('attributes'),
    items: [
      {
        id: 'add_cluster_info',
        label: 'Add Cluster Info',
        description: 'Add static cluster-scoped attributes to your data.',
        type: ActionsType.ADD_CLUSTER_INFO,
        icon: getActionIcon(ActionsType.ADD_CLUSTER_INFO),
        docsEndpoint: '/pipeline/actions/attributes/addclusterinfo',
        docsDescription: 'The “Add Cluster Info” Odigos Action can be used to add resource attributes to telemetry signals originated from the k8s cluster where the Odigos is running.',
        allowedSignals: ['TRACES', 'METRICS', 'LOGS'],
      },
      {
        id: 'delete_attribute',
        label: 'Delete Attribute',
        description: 'Delete attributes from logs, metrics, and traces.',
        type: ActionsType.DELETE_ATTRIBUTES,
        icon: getActionIcon(ActionsType.DELETE_ATTRIBUTES),
        docsEndpoint: '/pipeline/actions/attributes/deleteattribute',
        docsDescription: 'The “Delete Attribute” Odigos Action can be used to delete attributes from logs, metrics, and traces.',
        allowedSignals: ['TRACES', 'METRICS', 'LOGS'],
      },
      {
        id: 'rename_attribute',
        label: 'Rename Attribute',
        description: 'Rename attributes in logs, metrics, and traces.',
        type: ActionsType.RENAME_ATTRIBUTES,
        icon: getActionIcon(ActionsType.RENAME_ATTRIBUTES),
        docsEndpoint: '/pipeline/actions/attributes/rename-attribute',
        docsDescription:
          'The “Rename Attribute” Odigos Action can be used to rename attributes from logs, metrics, and traces. Different instrumentations might use different attribute names for similar information. This action let’s you to consolidate the names across your cluster.',
        allowedSignals: ['TRACES', 'METRICS', 'LOGS'],
      },
      {
        id: 'pii-masking',
        label: 'PII Masking',
        description: 'Mask PII data in your traces.',
        type: ActionsType.PII_MASKING,
        icon: getActionIcon(ActionsType.PII_MASKING),
        docsEndpoint: '/pipeline/actions/attributes/piimasking',
        docsDescription: 'The “PII Masking” Odigos Action can be used to mask PII data from span attribute values.',
        allowedSignals: ['TRACES'],
      },
    ],
  },
  {
    id: 'sampler',
    label: 'Samplers',
    icon: getActionIcon('sampler'),
    items: [
      {
        id: 'error-sampler',
        label: 'Error Sampler',
        description: 'Sample errors based on percentage.',
        type: ActionsType.ERROR_SAMPLER,
        icon: getActionIcon('sampler'),
        docsEndpoint: '/pipeline/actions/sampling/errorsampler',
        docsDescription: 'The “Error Sampler” Odigos Action is a Global Action that supports error sampling by filtering out non-error traces.',
        allowedSignals: ['TRACES'],
      },
      {
        id: 'probabilistic-sampler',
        label: 'Probabilistic Sampler',
        description: 'Sample traces based on percentage.',
        type: ActionsType.PROBABILISTIC_SAMPLER,
        icon: getActionIcon('sampler'),
        docsEndpoint: '/pipeline/actions/sampling/probabilisticsampler',
        docsDescription: 'The “Probabilistic Sampler” Odigos Action supports probabilistic sampling based on a configured sampling percentage applied to the TraceID.',
        allowedSignals: ['TRACES'],
      },
      {
        id: 'latency-action',
        label: 'Latency Sampler',
        description: 'Add latency to your traces.',
        type: ActionsType.LATENCY_SAMPLER,
        icon: getActionIcon('sampler'),
        docsEndpoint: '/pipeline/actions/sampling/latencysampler',
        docsDescription: 'The “Latency Sampler” Odigos Action is an Endpoint Action that samples traces based on their duration for a specific service and endpoint (HTTP route) filter.',
        allowedSignals: ['TRACES'],
      },
    ],
  },
];
