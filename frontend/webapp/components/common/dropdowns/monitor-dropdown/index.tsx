import React, { useMemo } from 'react';
import { useSourceCRUD } from '@/hooks';
import type { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';
import { MONITORS_OPTIONS } from '@/utils';

interface Props {
  value?: DropdownOption[];
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
}

export const MonitorDropdown: React.FC<Props> = ({ value, onSelect, onDeselect, ...props }) => {
  const metricsOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    MONITORS_OPTIONS.forEach(({ id, value }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value });
    });

    return options;
  }, []);

  return <Dropdown title='Monitors' placeholder='All' options={metricsOptions} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};
