import React, { useState } from 'react';
import { ActionState, ActionsType } from '@/types';
import DeleteAttributeYaml from './delete.attribute.yaml';
import { KeyvalText } from '@/design.system';
import styled from 'styled-components';
import AddClusterInfoYaml from './add.cluster.info.yaml';

import theme from '@/styles/palette';
import { Check, Copy, YamlIcon } from '@/assets/icons/app';

const CodeBlockWrapper = styled.p`
  display: flex;
  align-items: center;
  font-family: ${theme.font_family.primary};
  color: ${theme.text.light_grey};
  a {
    color: ${theme.text.secondary};
    text-decoration: none;
    cursor: pointer;
  }
`;
const TitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  max-width: 600px;
`;

const DescriptionWrapper = styled(TitleWrapper)`
  line-height: 1.3;
  margin: 10px 0 8px 0;
`;

interface ActionsYamlProps {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}

export function ActionsYaml({ data, onChange }: ActionsYamlProps) {
  const [copied, setCopied] = useState(false);
  const [echoCommand, setEchoCommand] = useState('');

  function renderYamlEditor() {
    switch (data.type) {
      case ActionsType.DELETE_ATTRIBUTES:
        return (
          <DeleteAttributeYaml
            data={data}
            onChange={onChange}
            setEchoCommand={setEchoCommand}
          />
        );
      case ActionsType.ADD_CLUSTER_INFO:
        return (
          <AddClusterInfoYaml
            data={data}
            onChange={onChange}
            setEchoCommand={setEchoCommand}
          />
        );
      default:
        return <></>;
    }
  }

  function handleCopy() {
    navigator.clipboard.writeText(echoCommand);
    setCopied(true);
    setTimeout(() => {
      setCopied(false);
    }, 3000);
  }

  return (
    <div>
      <TitleWrapper>
        <YamlIcon style={{ width: 20, height: 20 }} />
        <KeyvalText
          weight={600}
        >{`YAML Preview - ${data.type.toLowerCase()}.actions.odigos.io`}</KeyvalText>
      </TitleWrapper>

      <DescriptionWrapper>
        <KeyvalText size={14}>
          This is the YAML representation of the action you are creating. You
          can use this YAML to create the action using kubectl as well as the
          UI.
        </KeyvalText>
        <KeyvalText size={14}></KeyvalText>
      </DescriptionWrapper>
      <div style={{ width: 400 }}>{renderYamlEditor()}</div>
      <div style={{ overflowX: 'hidden' }}>
        <CodeBlockWrapper>
          {copied ? (
            <Check style={{ width: 18, height: 12 }} />
          ) : (
            <Copy style={{ width: 18, height: 16 }} />
          )}
          <a style={{ margin: '0 4px' }} onClick={handleCopy}>
            Click here
          </a>
          to copy as kubectl command.
        </CodeBlockWrapper>
      </div>
    </div>
  );
}
