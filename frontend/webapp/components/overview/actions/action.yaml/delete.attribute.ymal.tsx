import { YMLEditor } from '@/design.system';
import { ActionState } from '@/types';
import React, { useEffect, useMemo, useState } from 'react';

export default function DeleteAttributeYaml({ data }: { data: ActionState }) {
  const [actionName, setActionName] = useState('');
  const [yaml, setYaml] = useState({});
  useEffect(() => {
    console.log({ data });
    setYaml({
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
            actionName: data.actionName,
            attributeNamesToDelete: data.actionData?.attributeNamesToDelete,
            signals: data.selectedMonitors.map((monitor) => monitor.label),
            actionNote: data.actionNote,
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
      ],
      kind: 'List',
      metadata: {
        resourceVersion: '',
      },
    });
  }, [data]);

  if (Object.keys(yaml).length === 0) {
    return <></>;
  }

  return (
    <>
      <YMLEditor data={yaml} setData={() => {}} />
    </>
  );
}
