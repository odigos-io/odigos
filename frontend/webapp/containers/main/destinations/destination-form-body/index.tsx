import React, { Dispatch, SetStateAction, useEffect, useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { DestinationDynamicFields } from './dynamic-fields';
import { type ConnectionStatus, TestConnection } from './test-connection';
import { type DestinationInput, type DestinationTypeItem, type DynamicField } from '@/types';
import { Divider, Input, MonitorsCheckboxes, NOTIFICATION_TYPE, NotificationNote, SectionTitle, SIGNAL_TYPE } from '@odigos/ui-components';

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

export const DestinationFormBody = ({ isUpdate, destination, formData, formErrors, validateForm, handleFormChange, dynamicFields, setDynamicFields }: Props) => {
  const { supportedSignals, testConnectionSupported, displayName } = destination || {};

  const [autoFilled, setAutoFilled] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>();

  const autoFillCheckRef = useRef(false);

  useEffect(() => {
    if (!!dynamicFields.length && !autoFillCheckRef.current) {
      autoFillCheckRef.current = true;
      let didAutoFill = false;

      for (let i = 0; i < dynamicFields.length; i++) {
        const { required, value } = dynamicFields[i];

        if (required) {
          if (![undefined, null, ''].includes(value)) {
            didAutoFill = true;
          } else {
            didAutoFill = false;
            break;
          }
        }
      }

      setAutoFilled(didAutoFill);
    }
  }, [dynamicFields, isFormDirty]);

  const dirtyForm = () => {
    setIsFormDirty(true);
    setConnectionStatus(undefined);
  };

  const supportedMonitors = useMemo(() => {
    const { logs, metrics, traces } = supportedSignals || {};
    const arr: SIGNAL_TYPE[] = [];

    if (logs?.supported) arr.push(SIGNAL_TYPE.LOGS);
    if (metrics?.supported) arr.push(SIGNAL_TYPE.METRICS);
    if (traces?.supported) arr.push(SIGNAL_TYPE.TRACES);

    return arr;
  }, [supportedSignals]);

  const selectedMonitors = useMemo(() => {
    const { logs, metrics, traces } = formData['exportedSignals'] || {};
    const arr: SIGNAL_TYPE[] = [];

    if (logs) arr.push(SIGNAL_TYPE.LOGS);
    if (metrics) arr.push(SIGNAL_TYPE.METRICS);
    if (traces) arr.push(SIGNAL_TYPE.TRACES);

    return arr;
  }, [formData['exportedSignals']]);

  const handleSelectedSignals = (signals: SIGNAL_TYPE[]) => {
    dirtyForm();
    handleFormChange('exportedSignals', {
      logs: signals.includes(SIGNAL_TYPE.LOGS),
      metrics: signals.includes(SIGNAL_TYPE.METRICS),
      traces: signals.includes(SIGNAL_TYPE.TRACES),
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

          {(testConnectionSupported || autoFilled) && (
            <NotesWrapper>
              {testConnectionSupported && connectionStatus === NOTIFICATION_TYPE.ERROR && (
                <NotificationNote type={NOTIFICATION_TYPE.ERROR} message='Connection failed. Please check your input and try again.' />
              )}
              {testConnectionSupported && connectionStatus === NOTIFICATION_TYPE.SUCCESS && <NotificationNote type={NOTIFICATION_TYPE.SUCCESS} message='Connection succeeded.' />}
              {autoFilled && <NotificationNote type={NOTIFICATION_TYPE.DEFAULT} message={`Odigos autocompleted ${displayName} connection details.`} />}
            </NotesWrapper>
          )}

          <Divider />
        </>
      )}

      <MonitorsCheckboxes
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
};
