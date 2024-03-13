import React from 'react';
import { ActionState, ActionsType } from '@/types';
import DeleteAttributeYaml from './delete.attribute.ymal';

interface ActionsYamlProps {
  type: string;
  data: ActionState;
}

export function ActionsYaml({ type, data }: ActionsYamlProps) {
  function renderYamlEditor() {
    switch (type) {
      case ActionsType.DELETE_ATTRIBUTES:
        return <DeleteAttributeYaml data={data} />;
      default:
        return <></>;
    }
  }

  return <div>{renderYamlEditor()}</div>;
}
