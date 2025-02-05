import React, { useMemo } from 'react';
import { useSourceCRUD } from '@/hooks';
import { BACKEND_BOOLEAN } from '@/utils';
import { Dropdown, type DropdownProps } from '@odigos/ui-components';

interface Props {
  title?: string;
  value?: DropdownProps['options'];
  onSelect: (val: DropdownProps['options'][0]) => void;
  onDeselect: (val: DropdownProps['options'][0]) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

export const ErrorDropdown: React.FC<Props> = ({ title = 'Error Message', value, onSelect, onDeselect, ...props }) => {
  const { sources } = useSourceCRUD();

  const options = useMemo(() => {
    const payload: DropdownProps['options'] = [];

    sources.forEach(({ conditions }) => {
      conditions?.forEach(({ status, message }) => {
        if (status === BACKEND_BOOLEAN.FALSE && !payload.find((opt) => opt.id === message)) {
          payload.push({ id: message, value: message });
        }
      });
    });

    return payload;
  }, [sources]);

  return <Dropdown title={title} placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
