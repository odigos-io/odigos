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
      case ActionsType.RENAME_ATTRIBUTES:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${Object.keys(item?.spec?.renames).length} renamed attributes`}
          </KeyvalText>
        );
      case ActionsType.ERROR_SAMPLER:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${item?.spec?.fallback_sampling_ratio}% sampling ratio`}s
          </KeyvalText>
        );
      case ActionsType.PROBABILISTIC_SAMPLER:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${item?.spec?.sampling_percentage}% sampling ratio`}
          </KeyvalText>
        );
      case ActionsType.LATENCY_SAMPLER:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${item?.spec?.endpoints_filters.length} endpoints`}
          </KeyvalText>
        );
      case ActionsType.PII_MASKING:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {`${
              item?.spec?.piiCategories.length === 1
                ? '1 category'
                : `${item?.spec?.piiCategories.length} categories`
            }`}
          </KeyvalText>
        );
      default:
        return <div>{item.type}</div>;
    }
  }

  return <>{renderContentByType()}</>;
}
