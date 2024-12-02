import { safeJsonParse } from '@/utils';
import type { DataCardRow } from '@/reuseable-components';
import type { ActualDestination, DestinationDetailsResponse, ExportedSignals } from '@/types';

const buildMonitorsList = (exportedSignals: ExportedSignals): string =>
  Object.keys(exportedSignals)
    .filter((key) => exportedSignals[key])
    .join(', ');

const buildCard = (destination: ActualDestination, destinationTypeDetails: DestinationDetailsResponse['destinationTypeDetails']) => {
  const { exportedSignals, destinationType, fields } = destination;

  const arr: DataCardRow[] = [{ title: 'Destination', value: destinationType.displayName }, { title: 'Monitors', type: 'monitors', value: buildMonitorsList(exportedSignals) }, { type: 'divider' }];

  Object.entries(safeJsonParse<Record<string, string>>(fields, {})).map(([key, value]) => {
    const found = destinationTypeDetails?.fields?.find((field) => field.name === key);

    const { type } = safeJsonParse(found?.componentProperties, { type: '' });
    const secret = type === 'password' ? new Array(value.length).fill('•').join('') : '';

    arr.push({
      title: found?.displayName || key,
      value: secret || value,
    });
  });

  return arr;
};

export default buildCard;
