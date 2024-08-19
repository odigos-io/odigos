'use client';
import React, { useState, useEffect } from 'react';
import theme from '@/styles/palette';
import { useTestConnection, useKeyDown } from '@/hooks';
import { Field, SelectedDestination } from '@/types';
import { renderFields } from './dynamic.fields';
import {
  cleanObjectEmptyStringsValues,
  SETUP,
  stringifyNonStringValues,
} from '@/utils';
import { DestinationBody } from '@/containers/setup/connection/connection.section';
import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalInput,
  KeyvalLoader,
  KeyvalText,
} from '@/design.system';
import {
  CheckboxWrapper,
  ConnectionMonitorsWrapper,
  FieldWrapper,
  CreateDestinationButtonWrapper,
} from './create.connection.form.styled';

interface CreateConnectionFormProps {
  fields: Field[];
  destination: SelectedDestination;
  onSubmit: (formData: DestinationBody) => void;
  dynamicFieldsValues?: {
    [key: string]: any;
  };
  destinationNameValue?: string | null;
  checkboxValues?: {
    [key: string]: boolean;
  };
}

const MONITORS = [
  { id: 'logs', label: SETUP.MONITORS.LOGS, checked: true },
  { id: 'metrics', label: SETUP.MONITORS.METRICS, checked: true },
  { id: 'traces', label: SETUP.MONITORS.TRACES, checked: true },
];

// fields are the current destination supported fields which we want to have.
// fieldValues read the actual values that are received from the cluster.
// if there are field values which are not part of the current schema, we want to remove them.
const sanitizeDynamicFields = (
  fields: Field[],
  fieldValues: Record<string, any> | undefined
): Record<string, any> => {
  if (!fieldValues) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(fieldValues).filter(([key, value]) =>
      fields.find((field) => field.name === key)
    )
  );
};

export function CreateConnectionForm({
  fields,
  onSubmit,
  destination,
  dynamicFieldsValues,
  destinationNameValue,
  checkboxValues,
}: CreateConnectionFormProps) {
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);
  const [isCreateButtonDisabled, setIsCreateButtonDisabled] = useState(true);
  const [isConnectionTested, setIsConnectionTested] = useState({
    enabled: null,
    message: '',
  });
  const [dynamicFields, setDynamicFields] = useState(
    sanitizeDynamicFields(fields, dynamicFieldsValues)
  );
  const [destinationName, setDestinationName] = useState<string>(
    destinationNameValue || ''
  );

  const { testConnection } = useTestConnection();

  useEffect(() => {
    setInitialDynamicFields();
  }, [fields]);

  useEffect(() => {
    isFormValid();
  }, [destinationName, dynamicFields]);

  useEffect(() => {
    filterSupportedMonitors();
  }, [destination]);

  useKeyDown('Enter', handleKeyPress);

  function handleKeyPress(e: any) {
    if (!isCreateButtonDisabled) {
      onCreateClick();
    }
  }

  function setInitialDynamicFields() {
    if (fields) {
      const defaultValues = fields.reduce(
        (acc: { [key: string]: string }, field: Field) => {
          const value = dynamicFields[field.name] || field.initial_value || '';
          acc[field.name] = value;
          return acc;
        },
        {} as { [key: string]: string }
      );
      setDynamicFields(defaultValues);
    }
  }

  function filterSupportedMonitors() {
    const data: any = !checkboxValues
      ? MONITORS
      : MONITORS.map((monitor) => ({
          ...monitor,
          checked: checkboxValues[monitor.id],
        }));

    setSelectedMonitors(
      data.filter(({ id }) => destination?.supported_signals?.[id]?.supported)
    );
  }

  const handleCheckboxChange = (id: string) => {
    setSelectedMonitors((prevSelectedMonitors) => {
      const totalSelected = prevSelectedMonitors.filter(
        ({ checked }) => checked
      ).length;

      if (
        totalSelected === 1 &&
        prevSelectedMonitors.find((item) => item.id === id)?.checked
      ) {
        return prevSelectedMonitors; // Prevent unchecking the last selected checkbox
      }

      const updatedMonitors = prevSelectedMonitors.map((checkbox) =>
        checkbox.id === id
          ? { ...checkbox, checked: !checkbox.checked }
          : checkbox
      );

      return updatedMonitors;
    });
  };

  function handleDynamicFieldChange(name: string, value: string) {
    if (isConnectionTested.enabled !== null) {
      setIsConnectionTested({ enabled: null, message: '' });
    }

    setDynamicFields((prevFields) => ({ ...prevFields, [name]: value }));
  }

  function isFormValid() {
    let isValid = true;
    for (let field of fields) {
      if (field.component_properties.required) {
        const value = dynamicFields[field.name];
        if (value === undefined || value.trim() === '' || !destinationName) {
          isValid = false;
          break;
        }
      }
    }
    setIsCreateButtonDisabled(!isValid);
  }

  function onCreateClick() {
    const signals = selectedMonitors.reduce(
      (acc, { id, checked }) => ({ ...acc, [id]: checked }),
      {}
    );

    const stringifyFields = stringifyNonStringValues(dynamicFields);
    const fields = cleanObjectEmptyStringsValues(stringifyFields);

    const body = {
      name: destinationName,
      signals,
      fields,
      type: destination.type,
    };
    onSubmit(body);
  }

  async function handleCheckDestinationConnection() {
    const signals = selectedMonitors.reduce(
      (acc, { id, checked }) => ({ ...acc, [id]: checked }),
      {}
    );

    const stringifyFields = stringifyNonStringValues(dynamicFields);
    const fields = cleanObjectEmptyStringsValues(stringifyFields);

    const body = {
      name: destinationName,
      signals,
      fields,
      type: destination.type,
    };
    try {
      // testConnection(body);
    } catch (error) {}
  }

  return (
    <>
      <KeyvalText size={18} weight={600}>
        {dynamicFieldsValues
          ? SETUP.UPDATE_CONNECTION
          : SETUP.CREATE_CONNECTION}
      </KeyvalText>
      {selectedMonitors?.length >= 1 && (
        <ConnectionMonitorsWrapper>
          <KeyvalText size={14}>{SETUP.CONNECTION_MONITORS}</KeyvalText>
          <CheckboxWrapper>
            {selectedMonitors.map((checkbox) => (
              <KeyvalCheckbox
                key={checkbox?.id}
                value={checkbox?.checked}
                onChange={() => handleCheckboxChange(checkbox?.id)}
                label={checkbox?.label}
              />
            ))}
          </CheckboxWrapper>
        </ConnectionMonitorsWrapper>
      )}
      <FieldWrapper>
        <KeyvalInput
          label={SETUP.DESTINATION_NAME}
          value={destinationName}
          onChange={setDestinationName}
          required
        />
      </FieldWrapper>
      {renderFields(fields, dynamicFields, handleDynamicFieldChange)}
      <CreateDestinationButtonWrapper>
        {destination?.test_connection_supported && (
          <KeyvalButton
            variant="secondary"
            disabled={isCreateButtonDisabled}
            onClick={handleCheckDestinationConnection}
          >
            <div></div>
            {/* {isLoading ? (
              <KeyvalLoader width={9} height={9} />
            ) : isConnectionTested.enabled === null ? (
              <KeyvalText color={theme.text.secondary} size={14} weight={600}>
                {'Check Connection'}
              </KeyvalText>
            ) : isConnectionTested.enabled ? (
              <KeyvalText color={theme.colors.success} size={14} weight={600}>
                {'Connection Successful'}
              </KeyvalText>
            ) : (
              <KeyvalText color={theme.colors.error} size={14} weight={600}>
                {isConnectionTested.message}
              </KeyvalText>
            )} */}
          </KeyvalButton>
        )}
        <KeyvalButton
          disabled={
            isCreateButtonDisabled || isConnectionTested.enabled === false
          }
          onClick={onCreateClick}
        >
          <KeyvalText color={theme.colors.dark_blue} size={14} weight={600}>
            {dynamicFieldsValues
              ? SETUP.UPDATE_DESTINATION
              : SETUP.CREATE_DESTINATION}
          </KeyvalText>
        </KeyvalButton>
      </CreateDestinationButtonWrapper>
    </>
  );
}
