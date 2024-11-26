import { ActualDestination, DestinationDetailsResponse, ExportedSignals } from '@/types';
import { safeJsonParse } from '@/utils';

const buildMonitorsList = (exportedSignals: ExportedSignals): string =>
  Object.keys(exportedSignals)
    .filter((key) => exportedSignals[key])
    .join(', ') || 'N/A';

const buildCard = (destination: ActualDestination, destinationTypeDetails: DestinationDetailsResponse['destinationTypeDetails']) => {
  const { exportedSignals, destinationType, fields } = destination;

  const arr = [
    { title: 'Destination', value: destinationType.displayName || 'N/A' },
    { title: 'Monitors', value: buildMonitorsList(exportedSignals) },
  ];

  Object.entries(safeJsonParse<Record<string, string>>(fields, {})).map(([key, value]) => {
    const found = destinationTypeDetails?.fields?.find((field) => field.name === key);

    const { type } = safeJsonParse(found?.componentProperties, { type: '' });
    const secret = type === 'password' ? new Array(value.length).fill('•').join('') : '';

    arr.push({
      title: found?.displayName || key,
      value: secret || value || 'N/A',
    });
  });

  return arr;
};

export default buildCard;
