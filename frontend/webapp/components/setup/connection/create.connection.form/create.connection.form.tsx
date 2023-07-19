import React, { useState, useEffect } from "react";
import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalInput,
  KeyvalText,
} from "@/design.system";
import {
  CheckboxWrapper,
  ConnectionMonitorsWrapper,
  DynamicFieldsWrapper,
  FieldWrapper,
  CreateDestinationButtonWrapper,
} from "./create.connection.form.styled";
import { renderFields } from "./dynamic.fields";
import { SETUP } from "@/utils/constants";
import { DestinationBody } from "@/containers/setup/connection/connection.section";

export interface Field {
  name: string;
  component_type: string;
  display_name: string;
  component_properties: any;
  video_url: string;
}

interface CreateConnectionFormProps {
  fields: Field[];
  onSubmit: (formData: DestinationBody) => void;
  supportedSignals: {
    [key: string]: {
      supported: boolean;
    };
  };
}

const MONITORS = [
  { id: "logs", label: SETUP.MONITORS.LOGS, checked: true },
  { id: "metrics", label: SETUP.MONITORS.METRICS, checked: true },
  { id: "traces", label: SETUP.MONITORS.TRACES, checked: true },
];

export function CreateConnectionForm({
  fields,
  onSubmit,
  supportedSignals,
}: CreateConnectionFormProps) {
  const [destinationName, setDestinationName] = useState<string>("");
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);
  const [dynamicFields, setDynamicFields] = useState({});
  const [isCreateButtonDisabled, setIsCreateButtonDisabled] = useState(true);

  useEffect(() => {
    isFormValid();
  }, [destinationName, dynamicFields]);

  useEffect(() => {
    filterSupportedMonitors();
  }, [supportedSignals]);

  function filterSupportedMonitors() {
    setSelectedMonitors(
      MONITORS.filter(({ id }) => supportedSignals[id]?.supported)
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
    const dynamicFieldsValues = Object.values(dynamicFields);
    const isValid =
      !!destinationName &&
      dynamicFieldsValues.every((field: Field) => field) &&
      dynamicFieldsValues.length === fields?.length;

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
    <form action="javascript:void(0);">
      <KeyvalText size={18} weight={600}>
        {SETUP.CREATE_CONNECTION}
      </KeyvalText>
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
      <FieldWrapper>
        <KeyvalInput
          style={{ height: 36 }}
          label={SETUP.DESTINATION_NAME}
          value={destinationName}
          onChange={setDestinationName}
        />
      </FieldWrapper>
      <DynamicFieldsWrapper>
        {renderFields(fields, dynamicFields, handleDynamicFieldChange)}
      </DynamicFieldsWrapper>
      <CreateDestinationButtonWrapper>
        <KeyvalButton disabled={isCreateButtonDisabled} onClick={onCreateClick}>
          <KeyvalText color={"#203548"} size={14} weight={600}>
            {SETUP.CREATE_DESTINATION}
          </KeyvalText>
        </KeyvalButton>
      </CreateDestinationButtonWrapper>
    </form>
  );
}
