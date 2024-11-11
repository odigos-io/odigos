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

  const typesOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    sources.forEach(({ kind: id }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value: id });
    });

    return options;
  }, [sources]);

  return <Dropdown title='Type' placeholder='All' options={typesOptions} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};
