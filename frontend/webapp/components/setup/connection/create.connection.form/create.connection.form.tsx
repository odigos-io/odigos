import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalDropDown,
  KeyvalInput,
  KeyvalText,
} from "@/design.system";
import React, { useState, useEffect } from "react";
import {
  CheckboxWrapper,
  ConnectionMonitorsWrapper,
  DynamicFieldsWrapper,
  FieldWrapper,
  CreateDestinationButtonWrapper,
} from "./create.connection.form.styled";

const MONITORS = [
  { id: "logs", label: "Logs", checked: true },
  { id: "metrics", label: "Metrics", checked: true },
  { id: "traces", label: "Traces", checked: true },
];

export function CreateConnectionForm({ fields }: any) {
  const [destinationName, setDestinationName] = useState<string>("");
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);
  const [dynamicFields, setDynamicFields] = useState({});
  const [isCreateButtonDisabled, setIsCreateButtonDisabled] = useState(true);

  useEffect(() => {
    fields && mapFieldsToDynamicFields();
  }, [fields]);

  useEffect(() => {
    isFormValid();
  }, [destinationName, dynamicFields]);

  function mapFieldsToDynamicFields() {
    return fields?.reduce((acc, field) => {
      acc[field?.name] = null;
      return acc;
    }, {});
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
      dynamicFieldsValues.every((field) => field) &&
      dynamicFieldsValues.length === fields?.length;

    setIsCreateButtonDisabled(!isValid);
  }

  function renderFields() {
    return fields?.map((field) => {
      switch (field?.component_type) {
        case "input":
          return (
            <FieldWrapper key={field?.name}>
              <KeyvalInput
                style={{ height: 36 }}
                label={field?.display_name}
                value={dynamicFields[field?.name]}
                onChange={(value) =>
                  handleDynamicFieldChange(field?.name, value)
                }
                {...field?.component_properties}
              />
            </FieldWrapper>
          );
        case "dropdown":
          return (
            <>
              <FieldWrapper key={field?.name}>
                <KeyvalText size={14} weight={600} style={{ marginBottom: 8 }}>
                  {field?.display_name}
                </KeyvalText>
                <KeyvalDropDown
                  width={354}
                  data={field?.component_properties?.values.map((value) => {
                    return { label: value, id: value };
                  })}
                  onChange={({ label }) =>
                    handleDynamicFieldChange(field?.name, label)
                  }
                />
              </FieldWrapper>
            </>
          );
        default:
          return null;
      }
    });
  }

  return (
    <div>
      <KeyvalText size={18} weight={600}>
        {"Create connection"}
      </KeyvalText>
      <ConnectionMonitorsWrapper>
        <KeyvalText size={14}>{"This connection will monitor:"}</KeyvalText>
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
          label={"Destination Name"}
          value={destinationName}
          onChange={setDestinationName}
        />
      </FieldWrapper>
      <DynamicFieldsWrapper>{renderFields()}</DynamicFieldsWrapper>
      <CreateDestinationButtonWrapper>
        <KeyvalButton disabled={isCreateButtonDisabled}>
          <KeyvalText color={"#203548"} size={14} weight={600}>
            {"Create Destination"}
          </KeyvalText>
        </KeyvalButton>
      </CreateDestinationButtonWrapper>
    </div>
  );
}
