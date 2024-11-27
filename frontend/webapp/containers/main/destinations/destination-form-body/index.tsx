import React, { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { TestConnection } from './test-connection';
import { DestinationDynamicFields } from './dynamic-fields';
import type { DestinationInput, DestinationTypeItem, DynamicField } from '@/types';
import { CheckboxList, Divider, Input, NotificationNote, SectionTitle } from '@/reuseable-components';

interface Props {
  isUpdate?: boolean;
  destination?: DestinationTypeItem;
  isFormOk: boolean;
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

export function DestinationFormBody({ isUpdate, destination, isFormOk, formData, handleFormChange, dynamicFields, setDynamicFields }: Props) {
  const { supportedSignals, testConnectionSupported, displayName } = destination || {};

  const [isFormDirty, setIsFormDirty] = useState(false);
  const [showConnectionError, setShowConnectionError] = useState(false);

  // this is to allow test connection when there are default values loaded
  useEffect(() => {
    if (isFormOk) setIsFormDirty(true);
  }, [isFormOk]);

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
            description={`Connect ${displayName} destination with Odigos.`}
            actionButton={
              testConnectionSupported && (
                <TestConnection
                  destination={formData}
                  disabled={!isFormOk || !isFormDirty}
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
          if (!isFormDirty) setIsFormDirty(true);
          handleFormChange(`exportedSignals.${signal}`, value);
        }}
      />

      {!isUpdate && (
        <Input
          title='Destination name'
          placeholder='Enter destination name'
          value={formData.name}
          onChange={(e) => {
            if (!isFormDirty) setIsFormDirty(true);
            handleFormChange('name', e.target.value);
          }}
        />
      )}

      <DestinationDynamicFields
        fields={dynamicFields}
        onChange={(name: string, value: any) => {
          if (!isFormDirty) setIsFormDirty(true);
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
