import React from 'react';
import { ActionsType } from '@/types';
import DeleteAttributeYaml from './delete.attribute.ymal';

interface ActionsYamlProps {
  type: string;
}

export function ActionsYaml({ type }: ActionsYamlProps) {
  function renderYamlEditor() {
    switch (type) {
      case ActionsType.DELETE_ATTRIBUTES:
        return <DeleteAttributeYaml />;
      default:
        return <></>;
    }
  }

  return <div>{renderYamlEditor()}</div>;
}
