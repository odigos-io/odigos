import React from 'react';
import { ActionState, ActionsType } from '@/types';
import DeleteAttributeYaml from './delete.attribute.yaml';

interface ActionsYamlProps {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}

export function ActionsYaml({ data, onChange }: ActionsYamlProps) {
  function renderYamlEditor() {
    switch (data.type) {
      case ActionsType.DELETE_ATTRIBUTES:
        return <DeleteAttributeYaml data={data} onChange={onChange} />;
      default:
        return <></>;
    }
  }

  return <div>{renderYamlEditor()}</div>;
}
