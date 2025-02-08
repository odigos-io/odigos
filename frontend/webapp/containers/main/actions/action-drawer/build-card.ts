import { DISPLAY_TITLES } from '@odigos/ui-utils';
import { type Action } from '@odigos/ui-containers';
import { DATA_CARD_FIELD_TYPES, DataCardFieldsProps } from '@odigos/ui-components';

const buildCard = (action: Action) => {
  const {
    type,
    spec: {
      actionName,
      notes,
      signals,
      disabled,

      clusterAttributes,
      attributeNamesToDelete,
      renames,
      piiCategories,
      fallbackSamplingRatio,
      samplingPercentage,
      endpointsFilters,
    },
  } = action;

  const arr: DataCardFieldsProps['data'] = [
    { title: DISPLAY_TITLES.TYPE, value: type },
    { type: DATA_CARD_FIELD_TYPES.ACTIVE_STATUS, title: DISPLAY_TITLES.STATUS, value: String(!disabled) },
    { title: DISPLAY_TITLES.NAME, value: actionName },
    { title: DISPLAY_TITLES.NOTES, value: notes },
    { type: DATA_CARD_FIELD_TYPES.DIVIDER, width: '100%' },
    { type: DATA_CARD_FIELD_TYPES.MONITORS, title: DISPLAY_TITLES.SIGNALS_FOR_PROCESSING, value: signals.map((str) => str.toLowerCase()).join(', ') },
  ];

  if (clusterAttributes) {
    let str = '';
    clusterAttributes.forEach(({ attributeName, attributeStringValue }, idx) => {
      str += `${attributeName}: ${attributeStringValue}`;
      if (idx < clusterAttributes.length - 1) str += ', ';
    });

    arr.push({ title: 'Attributes', value: str });
  }

  if (attributeNamesToDelete) {
    let str = '';
    attributeNamesToDelete.forEach((attributeName, idx) => {
      str += attributeName;
      if (idx < attributeNamesToDelete.length - 1) str += ', ';
    });

    arr.push({ title: 'Attributes', value: str });
  }

  if (renames) {
    let str = '';
    const entries = Object.entries(renames);
    entries.forEach(([oldName, newName], idx) => {
      str += `${oldName}: ${newName}`;
      if (idx < entries.length - 1) str += ', ';
    });

    arr.push({ title: 'Attributes', value: str });
  }

  if (piiCategories) {
    let str = '';
    piiCategories.forEach((attributeName, idx) => {
      str += attributeName;
      if (idx < piiCategories.length - 1) str += ', ';
    });

    arr.push({ title: 'Attributes', value: str });
  }

  if (fallbackSamplingRatio) {
    arr.push({ title: 'Sampling Ratio', value: String(fallbackSamplingRatio) });
  }

  if (samplingPercentage) {
    arr.push({ title: 'Sampling Percentage', value: samplingPercentage });
  }

  if (endpointsFilters) {
    endpointsFilters.forEach(({ serviceName, httpRoute, minimumLatencyThreshold, fallbackSamplingRatio }, idx) => {
      let str = '';
      str += `Service Name: ${serviceName}\n`;
      str += `HTTP Route: ${httpRoute}\n`;
      str += `Min. Latency: ${minimumLatencyThreshold}\n`;
      str += `Sampling Ratio: ${fallbackSamplingRatio}`;

      arr.push({ title: `Endpoint${endpointsFilters.length > 1 ? ` #${idx + 1}` : ''}`, value: str });
    });
  }

  return arr;
};

export default buildCard;
