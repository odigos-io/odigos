import { DISPLAY_TITLES, safeJsonParse } from '@/utils';
import { DataCardRow, DataCardFieldTypes } from '@/reuseable-components';
import type { ActualDestination, DestinationDetailsResponse, ExportedSignals } from '@/types';

const buildMonitorsList = (exportedSignals: ExportedSignals): string =>
  Object.keys(exportedSignals)
    .filter((key) => exportedSignals[key])
    .join(', ');

const buildCard = (destination: ActualDestination, destinationTypeDetails: DestinationDetailsResponse['destinationTypeDetails']) => {
  const { exportedSignals, destinationType, fields } = destination;

  const arr: DataCardRow[] = [
    { title: DISPLAY_TITLES.DESTINATION, value: destinationType.displayName },
    { type: DataCardFieldTypes.MONITORS, title: DISPLAY_TITLES.MONITORS, value: buildMonitorsList(exportedSignals) },
    { type: DataCardFieldTypes.DIVIDER, width: '100%' },
  ];

  Object.entries(safeJsonParse<Record<string, string>>(fields, {})).map(([key, value]) => {
    const found = destinationTypeDetails?.fields?.find((field) => field.name === key);

    const { type } = safeJsonParse(found?.componentProperties, { type: '' });
    const secret = type === 'password' ? new Array(11).fill('•').join('') : '';

    arr.push({
      title: found?.displayName || key,
      value: secret || value,
    });
  });

  return arr;
};

export default buildCard;
