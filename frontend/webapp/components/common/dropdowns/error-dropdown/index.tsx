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

export const ErrorDropdown: React.FC<Props> = ({ title = 'Error Message', value, onSelect, onDeselect, ...props }) => {
  const { sources } = useSourceCRUD();

  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    sources.forEach(({ instrumentedApplicationDetails: { conditions } }) => {
      conditions.forEach(({ type, status, message }) => {
        if (status === 'False' && !payload.find((opt) => opt.value === message)) {
          payload.push({ id: message, value: message });
        }
      });
    });

    return payload;
  }, [sources]);

  return <Dropdown title={title} placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
