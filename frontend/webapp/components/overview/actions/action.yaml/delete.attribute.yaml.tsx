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

export default function DeleteAttributeYaml({
  data,
  onChange,
}: {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}) {
  const [yaml, setYaml] = useState({});
  const [echoCommand, setEchoCommand] = useState('');
  const [copied, setCopied] = useState(false);

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
      - ${data.actionData?.attributeNamesToDelete.join('\n      - ')}`
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

  function handleCopy() {
    navigator.clipboard.writeText(echoCommand);
    setCopied(true);
    setTimeout(() => {
      setCopied(false);
    }, 3000);
  }

  if (Object.keys(yaml).length === 0) {
    return null;
  }

  return (
    <div style={{ width: 600, overflowX: 'hidden' }}>
      <YMLEditor data={yaml} setData={() => {}} />

      <CodeBlockWrapper>
        {copied ? (
          <Check style={{ width: 18, height: 12 }} />
        ) : (
          <YamlIcon style={{ width: 18, height: 18 }} />
        )}
        <a style={{ margin: '0 4px' }} onClick={handleCopy}>
          Click here
        </a>
        to copy as kubectl command.
      </CodeBlockWrapper>
    </div>
  );
}
