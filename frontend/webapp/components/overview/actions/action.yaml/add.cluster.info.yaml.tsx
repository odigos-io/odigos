import React, { useEffect, useState } from 'react';
import { ActionState } from '@/types';
import { YMLEditor } from '@/design.system';

export default function AddClusterInfoYaml({
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

    const clusterAttributes = data.actionData?.clusterAttributes.map(
      (attr) => ({
        [attr.attributeName]: attr.attributeStringValue,
      })
    );

    const newYaml = {
      apiVersion: 'actions.odigos.io/v1alpha1',
      kind: 'AddClusterInfo',
      metadata: {
        generateName: 'da-',
        namespace: 'odigos-system',
      },
      spec: {
        actionName: data.actionName || undefined,
        clusterAttributes,
        signals: data.selectedMonitors
          .filter((m) => m.checked)
          .map((m) => m.label.toUpperCase()),
        actionNote: data.actionNote || undefined,
      },
    };

    const echoCommand = `echo "
  apiVersion: actions.odigos.io/v1alpha1
  kind: AddClusterInfo
  metadata:
    generateName: da-
    namespace: odigos-system
  spec:
    ${data.actionName ? `actionName: ${data.actionName}` : ''}
    ${
      data.actionData?.clusterAttributes
        ? `clusterAttributes: 
      ${data.actionData.clusterAttributes
        .map((attr) => `  ${attr.attributeName}: ${attr.attributeStringValue}`)
        .join('\n      ')}`
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
    setYaml(newYaml);
  }, [data]);

  if (Object.keys(yaml).length === 0) {
    return null;
  }

  return <YMLEditor data={yaml} />;
}
