import { KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import { ActionData, ActionsType } from '@/types';
import React from 'react';

interface ActionRowDynamicContentProps {
  item: ActionData;
}

export default function ActionRowDynamicContent({
  item,
}: ActionRowDynamicContentProps) {
  function renderContentByType() {
    switch (item.type) {
      case ActionsType.ADD_CLUSTER_INFO:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${item?.spec.clusterAttributes.length} cluster attributes`}
          </KeyvalText>
        );
      case ActionsType.DELETE_ATTRIBUTES:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${item?.spec.attributeNamesToDelete.length} deleted attributes`}
          </KeyvalText>
        );
      default:
        return <div>{item.type}</div>;
    }
  }

  return <>{renderContentByType()}</>;
}
