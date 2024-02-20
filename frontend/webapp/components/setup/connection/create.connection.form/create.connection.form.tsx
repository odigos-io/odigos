import React, { useState, useEffect } from 'react';
import theme from '@/styles/palette';
import { useKeyDown } from '@/hooks';
import { SETUP } from '@/utils/constants';
import { Field } from '@/types/destinations';
import { renderFields } from './dynamic.fields';
import { DestinationBody } from '@/containers/setup/connection/connection.section';
import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalInput,
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
  onSubmit: (formData: DestinationBody) => void;
  supportedSignals: {
    [key: string]: {
      supported: boolean;
    };
  };
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
  supportedSignals,
  dynamicFieldsValues,
  destinationNameValue,
  checkboxValues,
}: CreateConnectionFormProps) {
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);
  const [isCreateButtonDisabled, setIsCreateButtonDisabled] = useState(true);
  const [dynamicFields, setDynamicFields] = useState(
    sanitizeDynamicFields(fields, dynamicFieldsValues)
  );
  const [destinationName, setDestinationName] = useState<string>(
    destinationNameValue || ''
  );

  useEffect(() => {
    setInitialDynamicFields();
  }, [fields]);

  useEffect(() => {
    isFormValid();
  }, [destinationName, dynamicFields]);

  useEffect(() => {
    filterSupportedMonitors();
  }, [supportedSignals]);

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
      data.filter(({ id }) => supportedSignals[id]?.supported)
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
    setDynamicFields((prevFields) => ({ ...prevFields, [name]: value }));
  }

  function isFormValid() {
    const areDynamicFieldsFilled = Object.values(dynamicFields).every(
      (field) => field
    );
    const isFieldLengthMatching =
      (fields?.length ?? 0) ===
      (dynamicFields ? Object.keys(dynamicFields).length : 0);

    const isValid =
      !!destinationName && areDynamicFieldsFilled && isFieldLengthMatching;

    setIsCreateButtonDisabled(!isValid);
  }

  function onCreateClick() {
    const signals = selectedMonitors.reduce(
      (acc, { id, checked }) => ({ ...acc, [id]: checked }),
      {}
    );
    const body = {
      name: destinationName,
      signals,
      fields: dynamicFields,
    };
    onSubmit(body);
  }

  return (
    <>
      <KeyvalText size={18} weight={600}>
        {dynamicFieldsValues
          ? SETUP.UPDATE_CONNECTION
          : SETUP.CREATE_CONNECTION}
      </KeyvalText>
      {selectedMonitors?.length > 1 && (
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
        <KeyvalButton disabled={isCreateButtonDisabled} onClick={onCreateClick}>
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
