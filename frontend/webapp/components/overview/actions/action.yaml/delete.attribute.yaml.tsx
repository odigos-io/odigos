import React, { useEffect, useState } from 'react';
import { ActionState } from '@/types';
import { YMLEditor } from '@/design.system';

export default function DeleteAttributeYaml({
  data,
  setEchoCommand,
}: {
  data: ActionState;
  setEchoCommand: (value: string) => void;
}) {
  const [yaml, setYaml] = useState({});

  useEffect(() => {
    if (data.actionName && data.actionName.endsWith(' ')) {
      return;
    }

    if (data.actionNote && data.actionNote.endsWith(' ')) {
      return;
    }

    if (data.actionData?.attributeNamesToDelete) {
      if (
        data.actionData.attributeNamesToDelete.some((attr) =>
          attr.endsWith(' ')
        )
      ) {
        return;
      }
    }

    const newYaml = {
      apiVersion: 'actions.odigos.io/v1alpha1',
      kind: 'DeleteAttribute',
      metadata: {
        generateName: 'da-',
        namespace: 'odigos-system',
      },
      spec: {
        actionName: data.actionName || undefined,
        attributeNamesToDelete: data.actionData?.attributeNamesToDelete,
        signals: data.selectedMonitors
          .filter((m) => m.checked)
          .map((m) => m.label.toUpperCase()),
        actionNote: data.actionNote || undefined,
      },
    };
    setYaml(newYaml);

    const echoCommand = `echo "
  apiVersion: actions.odigos.io/v1alpha1
  kind: DeleteAttribute
  metadata:
    generateName: da-
    namespace: odigos-system
  spec:
    ${data.actionName ? `actionName: ${data.actionName}` : ''}
    ${
      data.actionData?.attributeNamesToDelete
        ? `attributeNamesToDelete:
      - ${data.actionData?.attributeNamesToDelete
        .filter((attr) => attr !== '')
        .join('\n      - ')}`
        : ''
    }
    signals:
      - ${data.selectedMonitors
        .filter((m) => m.checked)
        .map((m) => m.label.toUpperCase())
        .join('\n      - ')}
    ${data.actionNote ? `actionNote: ${data.actionNote}` : ''}
  " | kubectl create -f -
                  
    `;
    setEchoCommand(echoCommand);
  }, [data]);

  if (Object.keys(yaml).length === 0) {
    return null;
  }

  return <YMLEditor data={yaml} />;
}
