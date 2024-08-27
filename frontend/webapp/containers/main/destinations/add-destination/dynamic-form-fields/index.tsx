import React from 'react';

import { INPUT_TYPES } from '@/utils/constants/string';
import {
  Dropdown,
  Input,
  TextArea,
  InputList,
  KeyValueInputsList,
} from '@/reuseable-components';

export function DynamicConnectDestinationFormFields({
  fields,
  onChange,
}: {
  fields: any[];
  onChange: (name: string, value: any) => void;
}) {
  console.log({ fields });
  return fields?.map((field: any) => {
    switch (field.componentType) {
      case INPUT_TYPES.INPUT:
        return (
          <Input
            key={field.name}
            {...field}
            onChange={(e) => onChange(field.name, e.target.value)}
          />
        );

      case INPUT_TYPES.DROPDOWN:
        return (
          <Dropdown
            key={field.name}
            {...field}
            onSelect={(option) =>
              onChange(field.name, { id: option.id, value: option.value })
            }
          />
        );
      case INPUT_TYPES.MULTI_INPUT:
        return (
          <InputList
            key={field.name}
            {...field}
            onChange={(value: string[]) =>
              onChange(field.name, JSON.stringify(value))
            }
          />
        );

      case INPUT_TYPES.KEY_VALUE_PAIR:
        return (
          <KeyValueInputsList
            key={field.name}
            {...field}
            onChange={(value) => onChange(field.name, JSON.stringify(value))}
          />
        );
      case INPUT_TYPES.TEXTAREA:
        return (
          <TextArea
            key={field.name}
            {...field}
            onChange={(e) => onChange(field.name, e.target.value)}
          />
        );
      default:
        return null;
    }
  });
}
