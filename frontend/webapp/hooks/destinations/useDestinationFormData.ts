import { useState, useEffect, useMemo, useCallback } from 'react';
import { safeJsonParse } from '@/utils';
import { useDrawerStore } from '@/store';
import { useQuery } from '@apollo/client';
import { useConnectDestinationForm } from '@/hooks';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import {
  DynamicField,
  ActualDestination,
  isActualDestination,
  DestinationDetailsResponse,
  SupportedDestinationSignals,
} from '@/types';

const DEFAULT_SUPPORTED_SIGNALS: SupportedDestinationSignals = {
  logs: { supported: false },
  metrics: { supported: false },
  traces: { supported: false },
};

export function useDestinationFormData() {
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);
  const [exportedSignals, setExportedSignals] = useState({
    logs: false,
    metrics: false,
    traces: false,
  });
  const [supportedSignals, setSupportedSignals] =
    useState<SupportedDestinationSignals>(DEFAULT_SUPPORTED_SIGNALS);

  const destination = useDrawerStore(({ selectedItem }) => selectedItem);
  const shouldSkip = !isActualDestination(destination?.item);
  const destinationType = isActualDestination(destination?.item)
    ? destination.item.destinationType.type
    : null;

  const { buildFormDynamicFields } = useConnectDestinationForm();

  const { data: destinationFields } = useQuery<DestinationDetailsResponse>(
    GET_DESTINATION_TYPE_DETAILS,
    { variables: { type: destinationType }, skip: shouldSkip }
  );

  // Memoize the buildFormDynamicFields to ensure it's stable across renders
  const memoizedBuildFormDynamicFields = useCallback(
    buildFormDynamicFields,
    []
  );

  useEffect(() => {
    if (destinationFields && isActualDestination(destination?.item)) {
      const { fields, exportedSignals, destinationType } = destination.item;
      const destinationTypeDetails = destinationFields.destinationTypeDetails;
      const formFields = memoizedBuildFormDynamicFields(
        destinationTypeDetails?.fields || []
      );
      const parsedFields = safeJsonParse<Record<string, string>>(fields, {});

      setDynamicFields(
        formFields.map((field) => ({
          ...field,
          value: parsedFields[field.name] || '',
        }))
      );

      setExportedSignals(exportedSignals);
      setSupportedSignals(destinationType.supportedSignals);
    }
  }, [destinationFields, destination, memoizedBuildFormDynamicFields]);

  const cardData = useMemo(() => {
    if (
      shouldSkip ||
      !isActualDestination(destination?.item) ||
      !destinationFields
    ) {
      return [
        { title: 'Error', value: 'No destination selected or data missing' },
      ];
    }

    const { exportedSignals, destinationType, fields } = destination.item;
    const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
    const destinationDetails = destinationFields.destinationTypeDetails?.fields;
    const fieldsData = buildDestinationFieldData(
      parsedFields,
      destinationDetails
    );

    return [
      { title: 'Destination', value: destinationType.displayName || 'N/A' },
      { title: 'Monitors', value: buildMonitorsList(exportedSignals) },
      ...fieldsData,
    ];
  }, [shouldSkip, destination, destinationFields]);

  return {
    cardData,
    dynamicFields,
    destinationType: destinationType || '',
    exportedSignals,
    supportedSignals,
    setExportedSignals,
    setDynamicFields,
  };
}

function buildDestinationFieldData(
  parsedFields: Record<string, string>,
  fieldDetails?: { name: string; displayName: string }[]
) {
  return Object.entries(parsedFields).map(([key, value]) => ({
    title:
      fieldDetails?.find((field) => field.name === key)?.displayName || key,
    value: value || 'N/A',
  }));
}

function buildMonitorsList(
  exportedSignals: ActualDestination['exportedSignals']
): string {
  return (
    Object.keys(exportedSignals)
      .filter((key) => exportedSignals[key] && key !== '__typename')
      .join(', ') || 'None'
  );
}
