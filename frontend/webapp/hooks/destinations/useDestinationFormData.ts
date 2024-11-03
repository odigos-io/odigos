import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
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
  DestinationDetailsField,
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
  const [supportedSignals, setSupportedSignals] = useState<SupportedDestinationSignals>(DEFAULT_SUPPORTED_SIGNALS);

  const destination = useDrawerStore(({ selectedItem }) => selectedItem);
  const shouldSkip = !isActualDestination(destination?.item);
  const destinationType = isActualDestination(destination?.item) ? destination.item.destinationType.type : null;

  const { buildFormDynamicFields } = useConnectDestinationForm();

  const { data: destinationFields } = useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
    variables: { type: destinationType },
    skip: shouldSkip,
  });

  // Memoize the buildFormDynamicFields to ensure it's stable across renders
  const memoizedBuildFormDynamicFields = useCallback(buildFormDynamicFields, []);

  const initialDynamicFieldsRef = useRef<DynamicField[]>([]);
  const initialExportedSignalsRef = useRef({
    logs: false,
    metrics: false,
    traces: false,
  });
  const initialSupportedSignalsRef = useRef<SupportedDestinationSignals>(DEFAULT_SUPPORTED_SIGNALS);

  useEffect(() => {
    if (destinationFields && isActualDestination(destination?.item)) {
      const { fields, exportedSignals, destinationType } = destination.item;
      const destinationTypeDetails = destinationFields.destinationTypeDetails;

      const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
      const formFields = memoizedBuildFormDynamicFields(destinationTypeDetails?.fields || []);

      const df = formFields.map((field) => {
        let fieldValue: any = parsedFields[field.name] || '';

        // Check if fieldValue is a JSON string that needs stringifying
        try {
          const parsedValue = JSON.parse(fieldValue);

          if (Array.isArray(parsedValue)) {
            // If it's an array, stringify it for setting the value
            fieldValue = parsedValue;
          }
        } catch (e) {
          // If parsing fails, it's not JSON, so we keep it as is
        }

        return {
          ...field,
          value: fieldValue,
        };
      });

      setDynamicFields(df);
      setExportedSignals(exportedSignals);
      setSupportedSignals(destinationType.supportedSignals);

      initialDynamicFieldsRef.current = df;
      initialExportedSignalsRef.current = exportedSignals;
      initialSupportedSignalsRef.current = destinationType.supportedSignals;
    }
  }, [destinationFields, destination, memoizedBuildFormDynamicFields]);

  const cardData = useMemo(() => {
    console.log('test', dynamicFields, destinationFields);

    if (shouldSkip || !isActualDestination(destination?.item) || !destinationFields) {
      return [{ title: 'Error', value: 'No destination selected or data missing' }];
    }

    const { exportedSignals, destinationType, fields } = destination.item;
    const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
    const destinationDetails = destinationFields.destinationTypeDetails?.fields;
    const fieldsData = buildDestinationFieldData(parsedFields, destinationDetails);

    return [
      { title: 'Destination', value: destinationType.displayName || 'N/A' },
      { title: 'Monitors', value: buildMonitorsList(exportedSignals) },
      ...fieldsData,
    ];
  }, [shouldSkip, destination, destinationFields]);

  // Reset function using initial values from refs
  const resetFormData = useCallback(() => {
    setDynamicFields(initialDynamicFieldsRef.current);
    setExportedSignals(initialExportedSignalsRef.current);
    setSupportedSignals(initialSupportedSignalsRef.current);
  }, []);

  return {
    cardData,
    dynamicFields,
    destinationType: destinationType || '',
    exportedSignals,
    supportedSignals,
    setExportedSignals,
    setDynamicFields,
    resetFormData,
  };
}

function buildDestinationFieldData(parsedFields: Record<string, string>, fieldDetails?: DestinationDetailsField[]) {
  return Object.entries(parsedFields).map(([key, value]) => {
    const found = fieldDetails?.find((field) => field.name === key);

    const { type } = safeJsonParse(found?.componentProperties, { type: '' });
    const secret = type === 'password' ? new Array(value.length).fill('*').join('') : '';

    return {
      title: found?.displayName || key,
      value: secret || value || 'N/A',
    };
  });
}

function buildMonitorsList(exportedSignals: ActualDestination['exportedSignals']): string {
  return (
    Object.keys(exportedSignals)
      .filter((key) => exportedSignals[key] && key !== '__typename')
      .join(', ') || 'None'
  );
}
