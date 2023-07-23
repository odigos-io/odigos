import React from "react";
import { KeyvalDropDown, KeyvalInput, KeyvalText } from "@/design.system";
import { FieldWrapper } from "./create.connection.form.styled";
import { INPUT_TYPES } from "@/utils/constants/string";
import { Field } from "./create.connection.form";

export function renderFields(
  fields: Field[],
  dynamicFields: object,
  onChange: (name: string, value: string) => void
) {
  return fields?.map((field) => {
    const { name, component_type, display_name, component_properties } = field;

    switch (component_type) {
      case INPUT_TYPES.INPUT:
        return (
          <FieldWrapper key={name}>
            <KeyvalInput
              style={{ height: 36 }}
              label={display_name}
              value={dynamicFields[name]}
              onChange={(value) => onChange(name, value)}
              {...component_properties}
            />
          </FieldWrapper>
        );
      case INPUT_TYPES.DROPDOWN:
        const dropdownData = component_properties?.values.map(
          (value: string) => ({
            label: value,
            id: value,
          })
        );
        return (
          <FieldWrapper key={name}>
            <KeyvalText size={14} weight={600} style={{ marginBottom: 8 }}>
              {display_name}
            </KeyvalText>
            <KeyvalDropDown
              width={354}
              data={dropdownData}
              onChange={({ label }) => onChange(name, label)}
            />
          </FieldWrapper>
        );
      default:
        return null;
    }
  });
}
