import type { ActionDataParsed } from '@/types';

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

  const arr = [
    { title: 'Type', value: type },
    { title: 'Status', value: String(!disabled) },
    { title: 'Monitors', value: signals.map((str) => str.toLowerCase()).join(', ') },
    { title: 'Name', value: actionName || 'N/A' },
    { title: 'Notes', value: notes || 'N/A' },
  ];

  if (clusterAttributes) {
    let str = '';
    clusterAttributes.forEach(({ attributeName, attributeStringValue }, idx) => {
      str += `${attributeName}: ${attributeStringValue}`;
      if (idx < clusterAttributes.length - 1) str += '\n';
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
