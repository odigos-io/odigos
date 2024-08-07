import React, { useEffect, useMemo, useState } from 'react';
import {
  DestinationDetailsResponse,
  DestinationTypeItem,
  DynamicField,
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
import { useConnectDestinationForm } from '@/hooks';
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
  const [checkedState, setCheckedState] = useState<boolean[]>([]);
  const [destinationName, setDestinationName] = useState<string>('');
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const { buildFormDynamicFields } = useConnectDestinationForm();

  const monitors = useMemo(() => {
    if (!destination) return [];

    const { logs, metrics, traces } = destination.supportedSignals;
    return [
      logs.supported && {
        id: 'logs',
        title: 'Logs',
      },
      metrics.supported && {
        id: 'metrics',
        title: 'Metrics',
      },
      traces.supported && {
        id: 'traces',
        title: 'Traces',
      },
    ].filter(Boolean);
  }, [destination]);

  useEffect(() => {
    data && console.log({ destination, data });

    if (data) {
      const df = buildFormDynamicFields(data.destinationTypeDetails.fields);
      console.log(
        'is missing fileds',
        df.length !== data.destinationTypeDetails.fields.length
      );
      console.log({ df });
      setDynamicFields(df);
    }
  }, [data]);

  function handleDynamicFieldChange(name: string, value: any) {
    setFormData((prev) => ({ ...prev, [name]: value }));
  }

  function handleSubmit() {
    console.log({ formData });
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
            checkedState={checkedState}
            setCheckedState={setCheckedState}
            monitors={monitors as []}
            title="This connection will monitor:"
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
