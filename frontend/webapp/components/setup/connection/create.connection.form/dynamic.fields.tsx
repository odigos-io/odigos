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
  TextArea,
} from '@/design.system';

const DEFAULT_KEY_VALUE_PAIR = {};
export function renderFields(
  fields: Field[],
  dynamicFields: object,
  onChange: (name: string, value: any) => void
) {
  return fields?.map((field) => {
    const { name, component_type, display_name, component_properties } = field;

    switch (component_type) {
      case INPUT_TYPES.INPUT:
        return (
          <FieldWrapper key={name}>
            <KeyvalInput
                data-cy={'create-destination-input-'+ name}
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
        let values = dynamicFields[name] || field.initial_value;
        if (typeof values === 'string') {
          values = safeJsonParse<string[]>(values, []);
        }
        return (
          <div key={name} style={{ marginTop: 22 }}>
            <MultiInputTable
              title={display_name}
              values={values}
              placeholder="Add value"
              onValuesChange={(value: string[]) =>
                onChange(name, value.length === 0 ? [] : value)
              }
              {...component_properties}
            />
          </div>
        );

      case INPUT_TYPES.KEY_VALUE_PAIR:
        let keyValues = dynamicFields[name] || DEFAULT_KEY_VALUE_PAIR;
        if (typeof keyValues === 'string') {
          keyValues = safeJsonParse<{ [key: string]: string }>(keyValues, {});
        }

        keyValues = Object.keys(keyValues).map((key) => ({
          key,
          value: keyValues[key],
        }));

        const array: KeyValue[] = [];
        let id = 0;
        for (const item of keyValues) {
          const { key, value } = item;
          array.push({ id: id++, key: key, value: value as string });
        }
        keyValues = array;

        return (
          <div key={name} style={{ marginTop: 22 }}>
            <KeyValuePair
              title={display_name}
              setKeyValues={(value) => {
                const data = value
                  .map((item) => {
                    return { key: item.key, value: item.value };
                  })
                  .reduce((obj, item) => {
                    obj[item.key] = item.value;
                    return obj;
                  }, {});
                onChange(name, data);
              }}
              keyValues={keyValues}
              {...component_properties}
            />
          </div>
        );
      case INPUT_TYPES.TEXTAREA:
        return (
          <FieldWrapper key={name} style={{ width: 362 }}>
            <TextArea
              label={display_name}
              value={dynamicFields[name]}
              onChange={(value) => onChange(name, value.target.value)}
              {...component_properties}
            />
          </FieldWrapper>
        );
      default:
        return null;
    }
  });
}
