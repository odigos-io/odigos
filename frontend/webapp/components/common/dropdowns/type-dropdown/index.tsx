import React, { useMemo } from 'react';
import { useSourceCRUD } from '@/hooks';
import type { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';

interface Props {
  value?: DropdownOption[];
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
}

export const TypeDropdown: React.FC<Props> = ({ value, onSelect, onDeselect, ...props }) => {
  const { sources } = useSourceCRUD();

  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    sources.forEach(({ kind: id }) => {
      if (!payload.find((opt) => opt.id === id)) {
        payload.push({ id, value: id });
      }
    });

    return payload;
  }, [sources]);

  return <Dropdown title='Type' placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};
