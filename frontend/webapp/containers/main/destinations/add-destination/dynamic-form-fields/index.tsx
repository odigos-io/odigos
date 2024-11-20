import React from 'react';
import { INPUT_TYPES } from '@/utils';
import { Dropdown, Input, TextArea, InputList, KeyValueInputsList } from '@/reuseable-components';

export function DynamicConnectDestinationFormFields({ fields, onChange }: { fields: any[]; onChange: (name: string, value: any) => void }) {
  return fields?.map((field: any) => {
    const { componentType, ...rest } = field;

    switch (componentType) {
      case INPUT_TYPES.INPUT:
        return <Input key={field.name} {...rest} onChange={(e) => onChange(field.name, e.target.value)} />;
      case INPUT_TYPES.DROPDOWN:
        return <Dropdown key={field.name} {...rest} onSelect={(option) => onChange(field.name, { id: option.id, value: option.value })} />;
      case INPUT_TYPES.MULTI_INPUT:
        return <InputList key={field.name} {...rest} onChange={(value: string[]) => onChange(field.name, JSON.stringify(value))} />;
      case INPUT_TYPES.KEY_VALUE_PAIR:
        return <KeyValueInputsList key={field.name} {...rest} onChange={(value) => onChange(field.name, JSON.stringify(value))} />;
      case INPUT_TYPES.TEXTAREA:
        return <TextArea key={field.name} {...rest} onChange={(e) => onChange(field.name, e.target.value)} />;
      default:
        return null;
    }
  });
}
