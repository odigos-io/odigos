import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { SideMenu } from '@/components';
import { useQuery } from '@apollo/client';
import { useDispatch } from 'react-redux';
import { addConfiguredDestination } from '@/store';
import { TestConnection } from '../test-connection';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { Body, Container, SideMenuWrapper } from '../styled';
import { useConnectDestinationForm, useConnectEnv } from '@/hooks';
import { DynamicConnectDestinationFormFields } from '../dynamic-form-fields';
import {
  StepProps,
  DynamicField,
  ExportedSignals,
  DestinationInput,
  DestinationTypeItem,
  DestinationDetailsResponse,
  ConfiguredDestination,
} from '@/types';
import {
  CheckboxList,
  Divider,
  Input,
  NotificationNote,
  SectionTitle,
} from '@/reuseable-components';

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

const FormContainer = styled.div`
  display: flex;
  width: 100%;
  max-width: 500px;
  flex-direction: column;
  gap: 24px;
  height: 443px;
  overflow-y: auto;
  padding-right: 16px;
  box-sizing: border-box;
  overflow: overlay;
  max-height: calc(100vh - 410px);
`;

const NotificationNoteWrapper = styled.div`
  margin-top: 24px;
`;

interface ConnectDestinationModalBodyProps {
  destination: DestinationTypeItem | undefined;
  onSubmitRef: React.MutableRefObject<(() => void) | null>;
}

export function ConnectDestinationModalBody({
  destination,
  onSubmitRef,
}: ConnectDestinationModalBodyProps) {
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [destinationName, setDestinationName] = useState<string>('');
  const [showConnectionError, setShowConnectionError] = useState(false);
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);
  const [exportedSignals, setExportedSignals] = useState<ExportedSignals>({
    logs: false,
    metrics: false,
    traces: false,
  });

  const dispatch = useDispatch();
  const { connectEnv } = useConnectEnv();
  const { buildFormDynamicFields } = useConnectDestinationForm();

  const { data } = useQuery<DestinationDetailsResponse>(
    GET_DESTINATION_TYPE_DETAILS,
    {
      variables: { type: destination?.type },
      skip: !destination,
    }
  );

  const monitors = useMemo(() => {
    if (!destination) return [];

    const { logs, metrics, traces } = destination.supportedSignals;

    setExportedSignals({
      logs: logs.supported,
      metrics: metrics.supported,
      traces: traces.supported,
    });

    return [
      logs.supported && { id: 'logs', title: 'Logs' },
      metrics.supported && { id: 'metrics', title: 'Metrics' },
      traces.supported && { id: 'traces', title: 'Traces' },
    ].filter(Boolean);
  }, [destination]);

  useEffect(() => {
    if (data) {
      const df = buildFormDynamicFields(data.destinationTypeDetails.fields);
      setDynamicFields(df);
    }
  }, [data]);

  useEffect(() => {
    // Assign handleSubmit to the onSubmitRef so it can be triggered externally
    onSubmitRef.current = handleSubmit;
  }, [formData, destinationName, exportedSignals]);

  function handleDynamicFieldChange(name: string, value: any) {
    setFormData((prev) => ({ ...prev, [name]: value }));
  }

  function handleSignalChange(signal: string, value: boolean) {
    setExportedSignals((prev) => ({ ...prev, [signal]: value }));
  }

  async function handleSubmit() {
    const fields = Object.entries(formData).map(([name, value]) => ({
      key: name,
      value,
    }));

    function storeConfiguredDestination() {
      const destinationTypeDetails = dynamicFields.map((field) => ({
        title: field.title,
        value: formData[field.name],
      }));

      destinationTypeDetails.unshift({
        title: 'Destination name',
        value: destinationName,
      });

      const storedDestination: ConfiguredDestination = {
        exportedSignals,
        destinationTypeDetails,
        type: destination?.type || '',
        imageUrl: destination?.imageUrl || '',
        category: destination?.category || '',
        displayName: destination?.displayName || '',
      };

      dispatch(addConfiguredDestination(storedDestination));
    }

    const body: DestinationInput = {
      name: destinationName,
      type: destination?.type || '',
      exportedSignals,
      fields,
    };
    await connectEnv(body, storeConfiguredDestination);
  }

  if (!destination) return null;

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} currentStep={2} />
      </SideMenuWrapper>

      <Body>
        <SectionTitle
          title="Create connection"
          description="Connect selected destination with Odigos."
          actionButton={
            destination.testConnectionSupported ? (
              <TestConnection // TODO: refactor this after add form validation
                onError={() => setShowConnectionError(true)}
                destination={{
                  name: destinationName,
                  type: destination?.type || '',
                  exportedSignals,
                  fields: Object.entries(formData).map(([name, value]) => ({
                    key: name,
                    value,
                  })),
                }}
              />
            ) : (
              <></>
            )
          }
        />
        {showConnectionError && (
          <NotificationNoteWrapper>
            <NotificationNote
              type="error"
              text={
                'Connection failed. Please check your input and try once again.'
              }
            />
          </NotificationNoteWrapper>
        )}
        <Divider margin="24px 0" />
        <FormContainer>
          <CheckboxList
            monitors={monitors as []}
            title="This connection will monitor:"
            exportedSignals={exportedSignals}
            handleSignalChange={handleSignalChange}
          />
          <Input
            title="Destination name"
            placeholder="Enter destination name"
            value={destinationName}
            onChange={(e) => setDestinationName(e.target.value)}
          />
          <DynamicConnectDestinationFormFields
            fields={dynamicFields}
            onChange={handleDynamicFieldChange}
          />
        </FormContainer>
      </Body>
    </Container>
  );
}
