import React, { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { SignalUppercase } from '@/utils';
import { TestConnection } from './test-connection';
import { DestinationDynamicFields } from './dynamic-fields';
import type { DestinationInput, DestinationTypeItem, DynamicField } from '@/types';
import { Divider, Input, MonitoringCheckboxes, NotificationNote, SectionTitle } from '@/reuseable-components';

interface Props {
  isUpdate?: boolean;
  destination?: DestinationTypeItem;
  formData: DestinationInput;
  formErrors: Record<string, string>;
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

export function DestinationFormBody({ isUpdate, destination, formData, formErrors, handleFormChange, dynamicFields, setDynamicFields }: Props) {
  const { supportedSignals, testConnectionSupported, displayName } = destination || {};
  const isFormOk = useMemo(() => !Object.keys(formErrors).length, [formErrors]);

  const [isFormDirty, setIsFormDirty] = useState(false);
  const [showConnectionError, setShowConnectionError] = useState(false);

  // this is to allow test connection when there are default values loaded
  useEffect(() => {
    if (isFormOk) setIsFormDirty(true);
  }, [isFormOk]);

  const supportedMonitors = useMemo(() => {
    const { logs, metrics, traces } = supportedSignals || {};
    const arr: SignalUppercase[] = [];

    if (logs?.supported) arr.push('LOGS');
    if (metrics?.supported) arr.push('METRICS');
    if (traces?.supported) arr.push('TRACES');

    return arr;
  }, [supportedSignals]);

  const selectedMonitors = useMemo(() => {
    const { logs, metrics, traces } = formData['exportedSignals'] || {};
    const arr: SignalUppercase[] = [];

    if (logs) arr.push('LOGS');
    if (metrics) arr.push('METRICS');
    if (traces) arr.push('TRACES');

    return arr;
  }, [formData['exportedSignals']]);

  const handleSelectedSignals = (signals: SignalUppercase[]) => {
    setIsFormDirty(true);
    handleFormChange('exportedSignals', {
      logs: signals.includes('LOGS'),
      metrics: signals.includes('METRICS'),
      traces: signals.includes('TRACES'),
    });
  };

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

      <MonitoringCheckboxes
        title={isUpdate ? '' : 'This connection will monitor:'}
        required
        allowedSignals={supportedMonitors}
        selectedSignals={selectedMonitors}
        setSelectedSignals={handleSelectedSignals}
        errorMessage={formErrors['exportedSignals']}
      />

      {!isUpdate && (
        <Input
          title='Destination name'
          placeholder='Enter destination name'
          value={formData['name']}
          onChange={(e) => {
            setIsFormDirty(true);
            handleFormChange('name', e.target.value);
          }}
          errorMessage={formErrors['name']}
        />
      )}

      <DestinationDynamicFields
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
        formErrors={formErrors}
      />
    </Container>
  );
}
