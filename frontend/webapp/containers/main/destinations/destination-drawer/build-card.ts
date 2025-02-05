import { DISPLAY_TITLES } from '@/utils';
import { type Destination } from '@odigos/ui-containers';
import { compareCondition, safeJsonParse } from '@odigos/ui-utils';
import { type DestinationDetailsResponse, type ExportedSignals } from '@/types';
import { DATA_CARD_FIELD_TYPES, type DataCardFieldsProps } from '@odigos/ui-components';

const buildMonitorsList = (exportedSignals: ExportedSignals): string =>
  Object.keys(exportedSignals)
    .filter((key) => exportedSignals[key as keyof ExportedSignals])
    .join(', ');

const buildCard = (destination: Destination, destinationTypeDetails?: DestinationDetailsResponse['destinationTypeDetails']) => {
  const { exportedSignals, destinationType, fields } = destination;

  const arr: DataCardFieldsProps['data'] = [
    { title: DISPLAY_TITLES.DESTINATION, value: destinationType.displayName },
    { type: DATA_CARD_FIELD_TYPES.MONITORS, title: DISPLAY_TITLES.MONITORS, value: buildMonitorsList(exportedSignals) },
    { type: DATA_CARD_FIELD_TYPES.DIVIDER, width: '100%' },
  ];

  const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
  const sortedParsedFields =
    destinationTypeDetails?.fields.map((field) => ({ key: field.name, value: parsedFields[field.name] ?? null })).filter((item) => item.value !== null) ||
    Object.entries(parsedFields).map(([key, value]) => ({ key, value }));

  sortedParsedFields.map(({ key, value }) => {
    const { displayName, secret, componentProperties, hideFromReadData, customReadDataLabels } = destinationTypeDetails?.fields?.find((field) => field.name === key) || {};

    const shouldHide = !!hideFromReadData?.length
      ? compareCondition(
          hideFromReadData,
          (destinationTypeDetails?.fields || []).map((field) => ({ name: field.name, value: parsedFields[field.name] ?? null })),
        )
      : false;

    if (!shouldHide) {
      const { type } = safeJsonParse(componentProperties, { type: '' });
      const isSecret = (secret || type === 'password') && !!value.length ? new Array(10).fill('•').join('') : '';

      if (!!customReadDataLabels?.length) {
        customReadDataLabels.forEach(({ condition, ...custom }) => {
          if (condition == value) {
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
