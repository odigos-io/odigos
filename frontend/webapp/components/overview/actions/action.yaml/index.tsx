import React from 'react';
import { ActionState, ActionsType } from '@/types';
import DeleteAttributeYaml from './delete.attribute.yaml';
import { KeyvalText } from '@/design.system';
import { YamlIcon } from '@/assets/icons/app';
import styled from 'styled-components';
import AddClusterInfoYaml from './add.cluster.info.yaml';

const TitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
`;

interface ActionsYamlProps {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}

export function ActionsYaml({ data, onChange }: ActionsYamlProps) {
  function renderYamlEditor() {
    switch (data.type) {
      case ActionsType.DELETE_ATTRIBUTES:
        return <DeleteAttributeYaml data={data} onChange={onChange} />;
      case ActionsType.ADD_CLUSTER_INFO:
        return <AddClusterInfoYaml data={data} onChange={onChange} />;
      default:
        return <></>;
    }
  }

  return (
    <div>
      <TitleWrapper>
        <YamlIcon style={{ width: 20, height: 20 }} />
        <KeyvalText
          weight={600}
        >{`YAML Preview - ${data.type.toLowerCase()}.actions.odigos.io`}</KeyvalText>
      </TitleWrapper>

      <TitleWrapper>
        <KeyvalText size={14}>
          This is the YAML representation of the action you are creating.
        </KeyvalText>
      </TitleWrapper>
      {renderYamlEditor()}
    </div>
  );
}
