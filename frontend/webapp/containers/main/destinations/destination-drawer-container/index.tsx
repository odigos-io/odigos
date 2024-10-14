import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useMemo,
  useState,
} from 'react';
import { safeJsonParse } from '@/utils';
import { useDrawerStore } from '@/store';
import { useQuery } from '@apollo/client';
import { CardDetails, DestinationForm } from '@/components';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import {
  ActualDestination,
  isActualDestination,
  DestinationDetailsResponse,
  ExportedSignals,
  DynamicField,
  SupportedDestinationSignals,
} from '@/types';
import styled from 'styled-components';
import { useConnectDestinationForm } from '@/hooks';

export type DestinationDrawerHandle = {
  getCurrentData: () => {
    name: string;
    type: string;
    exportedSignals: ExportedSignals;
    fields: { key: string; value: any }[];
  };
};

interface DestinationDrawerProps {
  isEditing: boolean;
}

const DEFAULT_SUPPORTED_SIGNALS = {
  logs: {
    supported: false,
  },
  metrics: {
    supported: false,
  },
  traces: {
    supported: false,
  },
};

const DestinationDrawer = forwardRef<
  DestinationDrawerHandle,
  DestinationDrawerProps
>(({ isEditing }, ref) => {
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);
  const destination = useDrawerStore(({ selectedItem }) => selectedItem);
  const [destinationName, setDestinationName] = useState<string>('');
  const [supportedSignals, setSupportedSignals] =
    useState<SupportedDestinationSignals>(DEFAULT_SUPPORTED_SIGNALS);
  const [exportedSignals, setExportedSignals] = useState<ExportedSignals>({
    logs: false,
    metrics: false,
    traces: false,
  });
  const shouldSkip = !isActualDestination(destination?.item);
  const destinationType = isActualDestination(destination?.item)
    ? destination.item.destinationType.type
    : null;

  const { buildFormDynamicFields } = useConnectDestinationForm();

  const { data: destinationFields, error } =
    useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
      variables: { type: destinationType },
      skip: shouldSkip,
    });

  useImperativeHandle(ref, () => ({
    getCurrentData: () => {
      const fields = processFormFields(dynamicFields);
      const newDestination = {
        name: destinationName,
        type: destination?.type || '',
        exportedSignals,
        fields,
      };
      return newDestination;
    },
  }));

  useEffect(initDynamicFields, [destinationFields, destination]);

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

  function initDynamicFields() {
    if (destinationFields && destination) {
      const df = buildFormDynamicFields(
        destinationFields.destinationTypeDetails.fields
      );

      const { fields, exportedSignals, name, destinationType } =
        destination.item as ActualDestination;
      const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
      const newDynamicFields = df.map((field) => {
        if (field?.name in parsedFields) {
          return {
            ...field,
            value:
              field.componentType === 'dropdown'
                ? {
                    id: parsedFields[field.name],
                    value: parsedFields[field.name],
                  }
                : parsedFields[field.name],
          };
        }
        return field;
      });
      setDestinationName(name);
      setExportedSignals(exportedSignals);
      setDynamicFields(newDynamicFields);
      setSupportedSignals(destinationType.supportedSignals);
    }
  }

  function handleSignalChange(signal: string, value: boolean) {
    setExportedSignals((prev) => ({ ...prev, [signal]: value }));
  }

  function handleDynamicFieldChange(name: string, value: any) {
    setDynamicFields((prev) => {
      return prev.map((field) => {
        if (field.name === name) {
          return { ...field, value };
        }
        return field;
      });
    });
  }

  return isEditing ? (
    <FormContainer>
      <DestinationForm
        dynamicFields={dynamicFields}
        destinationName={destinationName}
        exportedSignals={exportedSignals}
        supportedSignals={supportedSignals}
        setDestinationName={setDestinationName}
        handleSignalChange={handleSignalChange}
        handleDynamicFieldChange={handleDynamicFieldChange}
      />
    </FormContainer>
  ) : (
    <CardDetails data={cardData} />
  );
});

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

function processFormFields(dynamicFields) {
  function processFieldValue(field) {
    return field.componentType === 'dropdown' ? field.value.value : field.value;
  }

  // Prepare fields for the request body
  return dynamicFields.map((field) => ({
    key: field.name,
    value: processFieldValue(field),
  }));
}

const FormContainer = styled.div`
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 24px;
  height: 100%;
  overflow-y: auto;
  padding-right: 16px;
  box-sizing: border-box;
  overflow: overlay;
  max-height: calc(100vh - 220px);
`;
