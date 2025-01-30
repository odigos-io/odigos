import React, { useMemo } from 'react';
import { Dropdown, MONITORS_OPTIONS, type DropdownProps } from '@odigos/ui-components';

interface Props {
  title?: string;
  value?: DropdownProps['options'];
  onSelect: (val: DropdownProps['options'][0]) => void;
  onDeselect: (val: DropdownProps['options'][0]) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

export const MonitorDropdown: React.FC<Props> = ({ title = 'Monitors', value, onSelect, onDeselect, ...props }) => {
  const options = useMemo(() => {
    const payload: DropdownProps['options'] = [];

    MONITORS_OPTIONS.forEach(({ id, value }) => {
      if (!payload.find((opt) => opt.id === id)) payload.push({ id, value });
    });

    return payload;
  }, []);

  return <Dropdown title={title} placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
