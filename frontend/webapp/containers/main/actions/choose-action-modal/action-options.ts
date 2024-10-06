import { ActionsType } from '@/types';

export const ACTION_OPTIONS = [
  {
    id: 'add_cluster_info',
    label: 'Add Cluster Info',
    description: 'Add static cluster-scoped attributes to your data.',
    type: ActionsType.ADD_CLUSTER_INFO,
    icon: '/icons/actions/addclusterinfo.svg',
  },
  {
    id: 'delete_attribute',
    label: 'Delete Attribute',
    description: 'Delete attributes from logs, metrics, and traces.',
    type: ActionsType.DELETE_ATTRIBUTES,
    icon: '/icons/actions/deleteattribute.svg',
  },
  {
    id: 'rename_attribute',
    label: 'Rename Attribute',
    description: 'Rename attributes in logs, metrics, and traces.',
    type: ActionsType.RENAME_ATTRIBUTES,
    icon: '/icons/actions/renameattribute.svg',
  },
  {
    id: 'pii-masking',
    label: 'PII Masking',
    description: 'Mask PII data in your traces.',
    type: ActionsType.PII_MASKING,
    icon: '/icons/actions/piimasking.svg',
  },
  {
    id: 'sampler',
    label: 'Samplers',
    description: '',
    type: ActionsType.PROBABILISTIC_SAMPLER,
    icon: '/icons/actions/sampler.svg',
    items: [
      {
        id: 'error-sampler',
        label: 'Error Sampler',
        description: 'Sample errors based on percentage.',
        type: ActionsType.ERROR_SAMPLER,
      },
      {
        id: 'probabilistic-sampler',
        label: 'Probabilistic Sampler',
        description: 'Sample traces based on percentage.',
        type: ActionsType.PROBABILISTIC_SAMPLER,
      },
      {
        id: 'latency-action',
        label: 'Latency Action',
        description: 'Add latency to your traces.',
        type: ActionsType.LATENCY_SAMPLER,
      },
    ],
  },
];
