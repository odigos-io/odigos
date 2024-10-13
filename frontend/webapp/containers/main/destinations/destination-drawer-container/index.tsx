import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { useDrawerStore } from '@/store';
import { useQuery } from '@apollo/client';
import { CardDetails } from '@/components';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import {
  ActualDestination,
  isActualDestination,
  DestinationDetailsResponse,
} from '@/types';

const DestinationDrawer: React.FC = () => {
  const destination = useDrawerStore(({ selectedItem }) => selectedItem);

  const shouldSkip = !isActualDestination(destination?.item);
  const destinationType = isActualDestination(destination?.item)
    ? destination.item.destinationType.type
    : null;

  const { data: destinationFields, error } =
    useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
      variables: { type: destinationType },
      skip: shouldSkip,
    });

  const cardData = useMemo(() => {
    if (shouldSkip || !destination?.item || !destinationFields) {
      return [
        { title: 'Error', value: 'No destination selected or data missing' },
      ];
    }

    const { exportedSignals, destinationType, fields } =
      destination.item as ActualDestination;
    const { destinationTypeDetails } = destinationFields;

    const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
    const destinationFieldData = buildDestinationFieldData(
      parsedFields,
      destinationTypeDetails?.fields
    );

    const monitors = buildMonitorsList(exportedSignals);

    return [
      { title: 'Destination', value: destinationType.displayName || 'N/A' },
      { title: 'Monitors', value: monitors || 'None' },
      ...destinationFieldData,
    ];
  }, [shouldSkip, destination, destinationFields]);

  if (error) {
    console.error('Error fetching destination details:', error);
    return <p>Error loading destination details</p>;
  }

  return <CardDetails data={cardData} />;
};

export { DestinationDrawer };

// Helper function to build the destination field data array
function buildDestinationFieldData(
  parsedFields: Record<string, string>,
  fieldDetails?: { name: string; displayName: string }[]
) {
  return Object.entries(parsedFields).map(([key, value]) => {
    const displayName =
      fieldDetails?.find((field) => field.name === key)?.displayName || key;
    return { title: displayName, value: value || 'N/A' };
  });
}

function buildMonitorsList(
  exportedSignals: ActualDestination['exportedSignals']
): string {
  return (
    Object.entries(exportedSignals)
      .filter(([key, isEnabled]) => isEnabled && key !== '__typename')
      .map(([key]) => key)
      .join(', ') || 'None'
  );
}
