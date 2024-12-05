import { DISPLAY_TITLES } from '@/utils';
import type { ActionDataParsed } from '@/types';
import { DataCardFieldTypes, type DataCardRow } from '@/reuseable-components';

const buildCard = (action: ActionDataParsed) => {
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
      fallback_sampling_ratio,
      sampling_percentage,
      endpoints_filters,
    },
  } = action;

  const arr: DataCardRow[] = [
    { title: DISPLAY_TITLES.TYPE, value: type },
    { type: DataCardFieldTypes.ACTIVE_STATUS, title: DISPLAY_TITLES.STATUS, value: String(!disabled) },
    { title: DISPLAY_TITLES.NAME, value: actionName },
    { title: DISPLAY_TITLES.NOTES, value: notes },
    { type: DataCardFieldTypes.DIVIDER, width: '100%' },
    { type: DataCardFieldTypes.MONITORS, title: DISPLAY_TITLES.SIGNALS_FOR_PROCESSING, value: signals.map((str) => str.toLowerCase()).join(', ') },
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

  if (fallback_sampling_ratio) {
    arr.push({ title: 'Sampling Ratio', value: String(fallback_sampling_ratio) });
  }

  if (sampling_percentage) {
    arr.push({ title: 'Sampling Percentage', value: sampling_percentage });
  }

  if (endpoints_filters) {
    endpoints_filters.forEach(({ service_name, http_route, minimum_latency_threshold, fallback_sampling_ratio }, idx) => {
      let str = '';
      str += `Service Name: ${service_name}\n`;
      str += `HTTP Route: ${http_route}\n`;
      str += `Min. Latency: ${minimum_latency_threshold}\n`;
      str += `Sampling Ratio: ${fallback_sampling_ratio}`;

      arr.push({ title: `Endpoint${endpoints_filters.length > 1 ? ` #${idx + 1}` : ''}`, value: str });
    });
  }

  return arr;
};

export default buildCard;
