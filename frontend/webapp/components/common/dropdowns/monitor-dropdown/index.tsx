import React, { useMemo } from 'react';
import { MONITORS_OPTIONS } from '@/utils';
import type { DropdownOption } from '@/types';
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

export const MonitorDropdown: React.FC<Props> = ({ title = 'Monitors', value, onSelect, onDeselect, ...props }) => {
  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    MONITORS_OPTIONS.forEach(({ id, value }) => {
      if (!payload.find((opt) => opt.id === id)) {
        payload.push({ id, value });
      }
    });

    return payload;
  }, []);

  return <Dropdown title={title} placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
