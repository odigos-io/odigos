import React, { useMemo } from 'react';
import { useNamespace } from '@/hooks';
import type { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';

interface Props {
  value?: DropdownOption;
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
}

export const NamespaceDropdown: React.FC<Props> = ({ value, onSelect, onDeselect, ...props }) => {
  const { allNamespaces } = useNamespace();

  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    allNamespaces?.forEach(({ name: id }) => {
      if (!payload.find((opt) => opt.id === id)) {
        payload.push({ id, value: id });
      }
    });

    return payload;
  }, [allNamespaces]);

  return <Dropdown title='Namespace' placeholder='Select namespace' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};
