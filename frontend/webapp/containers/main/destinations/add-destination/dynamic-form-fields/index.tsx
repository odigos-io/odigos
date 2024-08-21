import React from 'react';

import { INPUT_TYPES } from '@/utils/constants/string';
import { Dropdown, Input, TextArea } from '@/reuseable-components';

export function DynamicConnectDestinationFormFields({
  fields,
  onChange,
}: {
  fields: any[];
  onChange: (name: string, value: any) => void;
}) {
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
            onSelect={(option) => onChange(field.name, option.value)}
          />
        );
      case INPUT_TYPES.MULTI_INPUT:
        return <div></div>;

      case INPUT_TYPES.KEY_VALUE_PAIR:
        return <div></div>;
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
