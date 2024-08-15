import React, { useEffect, useMemo, useState } from 'react';
import {
  DestinationDetailsResponse,
  DestinationInput,
  DestinationTypeItem,
  DynamicField,
  ExportedSignals,
  StepProps,
} from '@/types';
import { SideMenu } from '@/components';
import {
  Button,
  CheckboxList,
  Divider,
  Input,
  SectionTitle,
} from '@/reuseable-components';
import { Body, Container, SideMenuWrapper } from '../styled';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useQuery } from '@apollo/client';
import styled from 'styled-components';
import { DynamicConnectDestinationFormFields } from '../dynamic-form-fields';
import { useConnectDestinationForm, useConnectEnv } from '@/hooks';

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
`;

interface ConnectDestinationModalBodyProps {
  destination: DestinationTypeItem | undefined;
}

export function ConnectDestinationModalBody({
  destination,
}: ConnectDestinationModalBodyProps) {
  const { data } = useQuery<DestinationDetailsResponse>(
    GET_DESTINATION_TYPE_DETAILS,
    {
      variables: { type: destination?.type },
      skip: !destination,
    }
  );
  const [exportedSignals, setExportedSignals] = useState<ExportedSignals>({
    logs: false,
    metrics: false,
    traces: false,
  });
  const [destinationName, setDestinationName] = useState<string>('');
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const { buildFormDynamicFields } = useConnectDestinationForm();
  const { connectEnv, result, loading, error } = useConnectEnv();

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

    const body: DestinationInput = {
      name: destinationName,
      type: destination?.type || '',
      exportedSignals,
      fields,
    };
    await connectEnv(body);
  }

  if (!destination) return null;

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} />
      </SideMenuWrapper>

      <Body>
        <SectionTitle
          title="Create connection"
          description="Connect selected destination with Odigos."
          buttonText="Check connection"
          onButtonClick={() => {}}
        />
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
          <Button onClick={handleSubmit} disabled={!destinationName}>
            <span>CONNECT</span>
          </Button>
        </FormContainer>
      </Body>
    </Container>
  );
}
