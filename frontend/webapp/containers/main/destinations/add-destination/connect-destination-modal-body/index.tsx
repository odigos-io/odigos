import React, { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { TestConnection } from '../test-connection';
import { DynamicConnectDestinationFormFields } from '../dynamic-form-fields';
import type { DestinationInput, DestinationTypeItem, DynamicField } from '@/types';
import { CheckboxList, Divider, Input, NotificationNote, SectionTitle } from '@/reuseable-components';

interface ConnectDestinationModalBodyProps {
  isUpdate?: boolean;
  destination?: DestinationTypeItem;
  formData: DestinationInput;
  handleFormChange: (key: keyof DestinationInput | string, val: any) => void;
  dynamicFields: DynamicField[];
  setDynamicFields: Dispatch<SetStateAction<DynamicField[]>>;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0 4px;
`;

export function ConnectDestinationModalBody({ isUpdate, destination, formData, handleFormChange, dynamicFields, setDynamicFields }: ConnectDestinationModalBodyProps) {
  const { supportedSignals, testConnectionSupported, displayName } = destination || {};

  const [hasSomeFields, setHasSomeFields] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);
  const [showConnectionError, setShowConnectionError] = useState(false);

  useEffect(() => {
    const has = !!formData.fields.find((field) => !!field.value);
    setHasSomeFields(has);
    setIsFormDirty(has); // this is to allow test connection when there are default values loaded
  }, [formData.fields]);

  const supportedMonitors = useMemo(() => {
    const { logs, metrics, traces } = supportedSignals || {};
    const arr: { id: string; title: string }[] = [];

    if (logs?.supported) arr.push({ id: 'logs', title: 'Logs' });
    if (metrics?.supported) arr.push({ id: 'metrics', title: 'Metrics' });
    if (traces?.supported) arr.push({ id: 'traces', title: 'Traces' });

    return arr;
  }, [supportedSignals]);

  return (
    <Container>
      {!isUpdate && (
        <>
          <SectionTitle
            title='Create connection'
            description='Connect selected destination with Odigos.'
            actionButton={
              testConnectionSupported && (
                <TestConnection
                  destination={formData}
                  disabled={!hasSomeFields || !isFormDirty}
                  clearStatus={() => {
                    setIsFormDirty(false);
                    setShowConnectionError(false);
                  }}
                  onError={() => {
                    setIsFormDirty(false);
                    setShowConnectionError(true);
                  }}
                />
              )
            }
          />

          {testConnectionSupported && showConnectionError ? (
            <NotificationNote type='error' message='Connection failed. Please check your input and try again.' />
          ) : testConnectionSupported && !showConnectionError && !!displayName ? (
            <NotificationNote type='default' message={`Odigos autocompleted ${displayName} connection details.`} />
          ) : null}
          <Divider />
        </>
      )}

      <CheckboxList
        monitors={supportedMonitors}
        title={isUpdate ? '' : 'This connection will monitor:'}
        exportedSignals={formData.exportedSignals}
        handleSignalChange={(signal, value) => {
          setIsFormDirty(true);
          handleFormChange(`exportedSignals.${signal}`, value);
        }}
      />

      {!isUpdate && (
        <Input
          title='Destination name'
          placeholder='Enter destination name'
          value={formData.name}
          onChange={(e) => {
            setIsFormDirty(true);
            handleFormChange('name', e.target.value);
          }}
        />
      )}

      <DynamicConnectDestinationFormFields
        fields={dynamicFields}
        onChange={(name: string, value: any) => {
          setIsFormDirty(true);
          setDynamicFields((prev) => {
            const payload = [...prev];
            const foundIndex = payload.findIndex((field) => field.name === name);

            if (foundIndex !== -1) {
              payload[foundIndex] = { ...payload[foundIndex], value };
            }

            return payload;
          });
        }}
      />
    </Container>
  );
}
