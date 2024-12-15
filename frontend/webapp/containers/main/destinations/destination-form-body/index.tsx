import React, { Dispatch, SetStateAction, useMemo, useState } from 'react';
import styled from 'styled-components';
import { SignalUppercase } from '@/utils';
import { type ConnectionStatus, TestConnection } from './test-connection';
import { DestinationDynamicFields } from './dynamic-fields';
import { Divider, Input, MonitoringCheckboxes, NotificationNote, SectionTitle } from '@/reuseable-components';
import { NOTIFICATION_TYPE, type DestinationInput, type DestinationTypeItem, type DynamicField } from '@/types';

interface Props {
  isUpdate?: boolean;
  destination?: DestinationTypeItem;
  formData: DestinationInput;
  formErrors: Record<string, string>;
  validateForm: () => boolean;
  handleFormChange: (key: keyof DestinationInput, val: any) => void;
  dynamicFields: DynamicField[];
  setDynamicFields: Dispatch<SetStateAction<DynamicField[]>>;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0 4px;
`;

const NotesWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export function DestinationFormBody({ isUpdate, destination, formData, formErrors, validateForm, handleFormChange, dynamicFields, setDynamicFields }: Props) {
  const { supportedSignals, testConnectionSupported, displayName } = destination || {};

  const [isFormDirty, setIsFormDirty] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>();

  const dirtyForm = () => {
    setIsFormDirty(true);
    setConnectionStatus(undefined);
  };

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
    dirtyForm();
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
            description={`Connect ${displayName} with Odigos.`}
            actionButton={
              testConnectionSupported && (
                <TestConnection
                  destination={formData}
                  disabled={!isFormDirty}
                  status={connectionStatus}
                  onError={() => {
                    setIsFormDirty(false);
                    setConnectionStatus(NOTIFICATION_TYPE.ERROR);
                  }}
                  onSuccess={() => {
                    setIsFormDirty(false);
                    setConnectionStatus(NOTIFICATION_TYPE.SUCCESS);
                  }}
                  validateForm={validateForm}
                />
              )
            }
          />

          {testConnectionSupported && (
            <NotesWrapper>
              {connectionStatus === NOTIFICATION_TYPE.ERROR && <NotificationNote type={NOTIFICATION_TYPE.ERROR} message='Connection failed. Please check your input and try again.' />}
              {connectionStatus === NOTIFICATION_TYPE.SUCCESS && <NotificationNote type={NOTIFICATION_TYPE.SUCCESS} message='Connection succeeded.' />}
              {!connectionStatus && <NotificationNote type={NOTIFICATION_TYPE.DEFAULT} message={`Odigos autocompleted ${displayName} connection details.`} />}
            </NotesWrapper>
          )}

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
            dirtyForm();
            handleFormChange('name', e.target.value);
          }}
          errorMessage={formErrors['name']}
        />
      )}

      <DestinationDynamicFields
        fields={dynamicFields}
        onChange={(name: string, value: any) => {
          dirtyForm();
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
