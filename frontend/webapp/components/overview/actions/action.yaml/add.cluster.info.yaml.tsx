import React, { useEffect, useState } from 'react';
import { YMLEditor } from '@/design.system';
import { ActionState } from '@/types';

export default function AddClusterInfoYaml({
  data,
  onChange,
}: {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}) {
  const [yaml, setYaml] = useState({});

  useEffect(() => {
    if (data.actionName && data.actionName.endsWith(' ')) {
      return;
    }

    if (data.actionNote && data.actionNote.endsWith(' ')) {
      return;
    }

    const clusterAttributes = data.actionData?.clusterAttributes.map(
      (attr) => ({
        [attr.attributeName]: attr.attributeStringValue,
      })
    );

    const newYaml = {
      apiVersion: 'v1',
      items: [
        {
          apiVersion: 'actions.odigos.io/v1alpha1',
          kind: 'DeleteAttribute',
          metadata: {
            creationTimestamp: new Date().toISOString(),
            generateName: 'da-',
            generation: 1,
            name: 'da-25p5x',
            namespace: 'odigos-system',
            resourceVersion: '8521549',
            uid: '2da7ad5d-cc07-432b-8e94-f95f35bbfedb',
          },
          spec: {
            actionName: data.actionName,
            clusterAttributes,
            signals: data.selectedMonitors
              .filter((m) => m.checked)
              .map((m) => m.label),
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
    };
    setYaml(newYaml);
  }, [data]);

  if (Object.keys(yaml).length === 0) {
    return null;
  }

  return (
    <>
      <YMLEditor data={yaml} setData={() => {}} />
    </>
  );
}
