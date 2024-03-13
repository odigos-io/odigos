import React, { useEffect, useState } from 'react';
import { YMLEditor } from '@/design.system';
import { ActionState } from '@/types';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Check, YamlIcon } from '@/assets/icons/app';

const CodeBlockWrapper = styled.p`
  display: flex;
  align-items: center;
  font-family: Inter;
  color: ${theme.text.light_grey};
  a {
    color: ${theme.text.secondary};
    text-decoration: none;
    cursor: pointer;
  }
`;
export default function AddClusterInfoYaml({
  data,
  onChange,
  setEchoCommand,
}: {
  data: ActionState;
  setEchoCommand: (value: string) => void;
  onChange: (key: string, value: any) => void;
}) {
  const [yaml, setYaml] = useState({});
  const [copied, setCopied] = useState(false);
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

  return <YMLEditor data={yaml} setData={() => {}} />;
}
