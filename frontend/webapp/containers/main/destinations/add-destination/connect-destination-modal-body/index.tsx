import React, { useEffect, useLayoutEffect, useMemo, useState } from 'react';
import { useAppStore } from '@/store';
import { INPUT_TYPES } from '@/utils';
import { SideMenu } from '@/components';
import { useQuery } from '@apollo/client';
import { FormContainer } from './form-container';
import { TestConnection } from '../test-connection';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { Body, Container, SideMenuWrapper } from '../styled';
import { Divider, SectionTitle } from '@/reuseable-components';
import { ConnectionNotification } from './connection-notification';
import { useComputePlatform, useConnectDestinationForm, useConnectEnv, useDestinationFormData, useEditDestinationFormHandlers } from '@/hooks';
import { StepProps, DestinationInput, DestinationTypeItem, DestinationDetailsResponse, ConfiguredDestination } from '@/types';

const SIDE_MENU_DATA: StepProps[] = [
  {
    title: 'DESTINATIONS',
    state: 'finish',
    stepNumber: 1,
  },
  {
    title: 'CONNECTION',
    state: 'active',
    stepNumber: 2,
  },
];

interface ConnectDestinationModalBodyProps {
  destination: DestinationTypeItem | undefined;
  onSubmitRef: React.MutableRefObject<(() => void) | null>;
  onFormValidChange: (isValid: boolean) => void;
}

export function ConnectDestinationModalBody({ destination, onSubmitRef, onFormValidChange }: ConnectDestinationModalBodyProps) {
  const [destinationName, setDestinationName] = useState<string>('');
  const [showConnectionError, setShowConnectionError] = useState(false);

  const { dynamicFields, exportedSignals, setDynamicFields, setExportedSignals } = useDestinationFormData();

  const { connectEnv } = useConnectEnv();
  const { refetch } = useComputePlatform();
  const { buildFormDynamicFields } = useConnectDestinationForm();

  const { handleDynamicFieldChange, handleSignalChange } = useEditDestinationFormHandlers(setExportedSignals, setDynamicFields);

  const addConfiguredDestination = useAppStore(({ addConfiguredDestination }) => addConfiguredDestination);

  const { data } = useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
    variables: { type: destination?.type },
    skip: !destination,
  });

  useLayoutEffect(() => {
    if (!destination) return;
    const { logs, metrics, traces } = destination.supportedSignals;
    setExportedSignals({
      logs: logs.supported,
      metrics: metrics.supported,
      traces: traces.supported,
    });
  }, [destination, setExportedSignals]);

  useEffect(() => {
    if (data && destination) {
      const df = buildFormDynamicFields(data.destinationTypeDetails.fields);

      const newDynamicFields = df.map((field) => {
        if (destination.fields && field?.name in destination.fields) {
          return {
            ...field,
            value: destination.fields[field.name],
          };
        }
        return field;
      });

      setDynamicFields(newDynamicFields);
    }
  }, [data, destination]);

  useEffect(() => {
    // Assign handleSubmit to the onSubmitRef so it can be triggered externally
    onSubmitRef.current = handleSubmit;
  }, [dynamicFields, destinationName, exportedSignals]);

  useEffect(() => {
    const isFormValid = dynamicFields.every((field) => (field.required ? field.value : true));
    onFormValidChange(isFormValid);
  }, [dynamicFields]);

  const monitors = useMemo(() => {
    if (!destination) return [];
    const { logs, metrics, traces } = destination.supportedSignals;

    return [
      logs.supported && { id: 'logs', title: 'Logs' },
      metrics.supported && { id: 'metrics', title: 'Metrics' },
      traces.supported && { id: 'traces', title: 'Traces' },
    ].filter(Boolean);
  }, [destination]);

  function onDynamicFieldChange(name: string, value: any) {
    setShowConnectionError(false);
    handleDynamicFieldChange(name, value);
  }
  function processFieldValue(field) {
    return field.componentType === INPUT_TYPES.DROPDOWN ? field.value.value : field.value;
  }

  function processFormFields() {
    // Prepare fields for the request body
    return dynamicFields.map((field) => ({
      key: field.name,
      value: processFieldValue(field),
    }));
  }

  async function handleSubmit() {
    // Prepare fields for the request body
    const fields = processFormFields();

    // Function to store configured destination to display in the UI
    function storeConfiguredDestination() {
      const destinationTypeDetails = dynamicFields.map((field) => ({
        title: field.title,
        value: processFieldValue(field),
      }));

      // Add 'Destination name' as the first item
      destinationTypeDetails.unshift({
        title: 'Destination name',
        value: destinationName,
      });

      // Construct the configured destination object
      const storedDestination: ConfiguredDestination = {
        exportedSignals,
        destinationTypeDetails,
        type: destination?.type || '',
        imageUrl: destination?.imageUrl || '',
        category: '', // Could be handled in a more dynamic way if needed
        displayName: destination?.displayName || '',
      };

      // Dispatch action to store the destination
      addConfiguredDestination(storedDestination);
    }

    // Prepare the request body
    const body: DestinationInput = {
      name: destinationName,
      type: destination?.type || '',
      exportedSignals,
      fields,
    };

    try {
      // Await connection and store the configured destination if successful
      // await connectEnv(body, storeConfiguredDestination);
      await connectEnv(body, refetch);
    } catch (error) {
      console.error('Failed to submit destination configuration:', error);
      // Handle error (e.g., show notification or alert)
    }
  }

  const actionButton = useMemo(() => {
    if (!!destination?.testConnectionSupported) {
      return (
        <TestConnection
          onError={() => {
            setShowConnectionError(true);
            onFormValidChange(false);
          }}
          destination={{
            name: destinationName,
            type: destination?.type || '',
            exportedSignals,
            fields: processFormFields(),
          }}
        />
      );
    }
    return null;
  }, [destination, destinationName, exportedSignals, processFormFields, onFormValidChange]);

  if (!destination) return null;

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} currentStep={2} />
      </SideMenuWrapper>

      <Body>
        <SectionTitle title='Create connection' description='Connect selected destination with Odigos.' actionButton={actionButton} />
        <ConnectionNotification showConnectionError={showConnectionError} destination={destination} />
        <Divider margin='24px 0' />
        <FormContainer
          monitors={monitors}
          dynamicFields={dynamicFields}
          exportedSignals={exportedSignals}
          destinationName={destinationName}
          handleDynamicFieldChange={onDynamicFieldChange}
          handleSignalChange={handleSignalChange}
          setDestinationName={setDestinationName}
        />
      </Body>
    </Container>
  );
}
