import React, { useMemo } from 'react';
import { useSourceCRUD } from '@/hooks';
import { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';

interface Props {
  title?: string;
  value?: DropdownOption[];
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

export const LanguageDropdown: React.FC<Props> = ({ title = 'Programming Languages', value, onSelect, onDeselect, ...props }) => {
  const { sources } = useSourceCRUD();

  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    sources.forEach(({ containers }) => {
      containers.forEach(({ language }) => {
        if (!payload.find((opt) => opt.id === language)) {
          payload.push({ id: language, value: language });
        }
      });
    });

    return payload.sort((a, b) => a.id.localeCompare(b.id));
  }, [sources]);

  return <Dropdown title={title} placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
