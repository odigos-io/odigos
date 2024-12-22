import { DISPLAY_TITLES, safeJsonParse } from '@/utils';
import { DataCardRow, DataCardFieldTypes } from '@/reuseable-components';
import type { ActualDestination, DestinationDetailsResponse, ExportedSignals } from '@/types';

const buildMonitorsList = (exportedSignals: ExportedSignals): string =>
  Object.keys(exportedSignals)
    .filter((key) => exportedSignals[key])
    .join(', ');

const buildCard = (destination: ActualDestination, destinationTypeDetails?: DestinationDetailsResponse['destinationTypeDetails']) => {
  const { exportedSignals, destinationType, fields } = destination;

  const arr: DataCardRow[] = [
    { title: DISPLAY_TITLES.DESTINATION, value: destinationType.displayName },
    { type: DataCardFieldTypes.MONITORS, title: DISPLAY_TITLES.MONITORS, value: buildMonitorsList(exportedSignals) },
    { type: DataCardFieldTypes.DIVIDER, width: '100%' },
  ];

  const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
  const sortedParsedFields =
    destinationTypeDetails?.fields.map((field) => ({ key: field.name, value: parsedFields[field.name] ?? null })).filter((item) => item.value !== null) ||
    Object.entries(parsedFields).map(([key, value]) => ({ key, value }));

  sortedParsedFields.map(({ key, value }) => {
    const { displayName, secret, componentProperties, hideFromReadData, customReadDataLabels } = destinationTypeDetails?.fields?.find((field) => field.name === key) || {};

    if (!hideFromReadData) {
      const { type } = safeJsonParse(componentProperties, { type: '' });
      const isSecret = secret || type === 'password' ? new Array(10).fill('•').join('') : '';

      if (!!customReadDataLabels?.length) {
        customReadDataLabels.forEach(({ condition, ...custom }) => {
          if (String(condition) == value) {
            arr.push({
              title: custom.title,
              value: custom.value,
            });
          }
        });
      } else {
        arr.push({
          title: displayName || key,
          value: isSecret || value,
        });
      }
    }
  });

  return arr;
};

export default buildCard;
