import React from 'react';
import { Field } from '@/types/destinations';
import { INPUT_TYPES } from '@/utils/constants/string';
import { FieldWrapper } from './create.connection.form.styled';
import {
  KeyvalDropDown,
  KeyvalInput,
  KeyvalText,
  MultiInput,
} from '@/design.system';

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

        const dropDownValue = dynamicFields[name]
          ? { id: dynamicFields[name], label: dynamicFields[name] }
          : null;

        return (
          <FieldWrapper key={name}>
            <KeyvalText size={14} weight={600} style={{ marginBottom: 8 }}>
              {display_name}
            </KeyvalText>
            <KeyvalDropDown
              width={354}
              data={dropdownData}
              onChange={({ label }) => onChange(name, label)}
              value={dropDownValue}
            />
          </FieldWrapper>
        );
      case INPUT_TYPES.MULTI_INPUT:
        const selectedData = dynamicFields[name]
          ? JSON.parse(dynamicFields[name])
          : null;

        const initialList =
          selectedData || JSON.parse(field?.initial_value || '');

        return (
          <FieldWrapper key={name}>
            <KeyvalText size={14} weight={600} style={{ marginBottom: 8 }}>
              {display_name}
            </KeyvalText>
            <MultiInput
              initialList={initialList}
              label={display_name}
              onListChange={(value: string[]) =>
                onChange(name, value.length === 0 ? '' : JSON.stringify(value))
              }
              {...component_properties}
            />
          </FieldWrapper>
        );
      default:
        return null;
    }
  });
}
