import { YMLEditor } from '@/design.system';
import React from 'react';

export default function DeleteAttributeYaml() {
  const data = {
    apiVersion: 'v1',
    items: [
      {
        apiVersion: 'actions.odigos.io/v1alpha1',
        kind: 'DeleteAttribute',
        metadata: {
          creationTimestamp: '2024-03-12T13:30:36Z',
          generateName: 'da-',
          generation: 1,
          name: 'da-25p5x',
          namespace: 'odigos-system',
          resourceVersion: '8521549',
          uid: '2da7ad5d-cc07-432b-8e94-f95f35bbfedb',
        },
        spec: {
          actionName: 'gjhg',
          attributeNamesToDelete: ['jghjgh'],
          signals: ['LOGS', 'METRICS', 'TRACES'],
        },
        status: {
          conditions: [
            {
              lastTransitionTime: '2024-03-12T13:30:36Z',
              message:
                'The action has been reconciled to a processor resource.',
              observedGeneration: 1,
              reason: 'ProcessorCreated',
              status: 'True',
              type: 'TransformedToProcessor',
            },
          ],
        },
      },
      {
        apiVersion: 'actions.odigos.io/v1alpha1',
        kind: 'DeleteAttribute',
        metadata: {
          creationTimestamp: '2024-03-12T13:30:01Z',
          generateName: 'da-',
          generation: 5,
          name: 'da-8258w',
          namespace: 'odigos-system',
          resourceVersion: '8529855',
          uid: '20409e13-d963-4ff3-8a24-462b6c8b315a',
        },
        spec: {
          actionName: 'Delete Attribute',
          attributeNamesToDelete: ['fdgd', 'Delete Attribute'],
          signals: ['LOGS', 'METRICS', 'TRACES'],
        },
        status: {
          conditions: [
            {
              lastTransitionTime: '2024-03-12T13:30:01Z',
              message:
                'The action has been reconciled to a processor resource.',
              observedGeneration: 5,
              reason: 'ProcessorCreated',
              status: 'True',
              type: 'TransformedToProcessor',
            },
          ],
        },
      },
    ],
    kind: 'List',
    metadata: {
      resourceVersion: '',
    },
  };

  return (
    <>
      <YMLEditor data={data} setData={() => {}} />
    </>
  );
}
