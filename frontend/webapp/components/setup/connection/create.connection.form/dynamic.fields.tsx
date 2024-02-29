import React from 'react';
import { Field } from '@/types/destinations';
import { KeyValue } from '@keyval-dev/design-system';
import { INPUT_TYPES } from '@/utils/constants/string';
import { safeJsonParse } from '@/utils/functions/strings';
import { FieldWrapper } from './create.connection.form.styled';
import {
  KeyvalDropDown,
  KeyvalInput,
  KeyValuePair,
  MultiInputTable,
} from '@/design.system';

const DEFAULT_KEY_VALUE_PAIR = [
  {
    id: 0,
    key: '',
    value: '',
  },
];

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
            <KeyvalDropDown
              label={display_name}
              width={354}
              data={dropdownData}
              onChange={({ label }) => onChange(name, label)}
              value={dropDownValue}
              {...component_properties}
            />
          </FieldWrapper>
        );
      case INPUT_TYPES.MULTI_INPUT:
        // const userInputData = safeJsonParse<string[] | null>(
        //   dynamicFields[name],
        //   null
        // );

        // // Use safeJsonParse to parse field?.initial_value, defaulting to an empty string if not available.
        // // This assumes that the initial value is supposed to be a string when parsed successfully.
        // // Adjust the fallback value as necessary to match the expected type
        // const initialList =
        //   userInputData || safeJsonParse<string[]>(field?.initial_value, []);

        const values = safeJsonParse<string[]>(dynamicFields[name], []);

        return (
          <div key={name} style={{ marginTop: 22 }}>
            <MultiInputTable
              title={display_name}
              values={values}
              placeholder="Add value"
              onValuesChange={(value: string[]) =>
                onChange(name, value.length === 0 ? '' : JSON.stringify(value))
              }
              {...component_properties}
            />
          </div>
        );

      case INPUT_TYPES.KEY_VALUE_PAIR:
        let keyValues: KeyValue[] = safeJsonParse<KeyValue[]>(
          dynamicFields[name],
          DEFAULT_KEY_VALUE_PAIR
        );

        if (dynamicFields[name] === '') {
          onChange(name, stringifyKeyValues(keyValues));
        }

        if (!Array.isArray(keyValues)) {
          //data return as json from server
          const array: KeyValue[] = [];
          let id = 0;
          for (const [key, value] of Object.entries(keyValues)) {
            array.push({ id: id++, key: key, value: value as string });
          }
          keyValues = array;
        }

        return (
          <div key={name} style={{ marginTop: 22 }}>
            <KeyValuePair
              title={display_name}
              setKeyValues={(value) => {
                onChange(name, stringifyKeyValues(value));
              }}
              keyValues={keyValues}
              {...component_properties}
            />
          </div>
        );
      default:
        return null;
    }
  });
}

function stringifyKeyValues(keyValues: KeyValue[]) {
  const resultMap = {};
  keyValues.forEach((item) => {
    const { key, value } = item;
    resultMap[key] = value;
  });
  return JSON.stringify(resultMap);
}
