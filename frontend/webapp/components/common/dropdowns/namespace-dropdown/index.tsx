import React, { useMemo } from 'react';
import { useNamespace } from '@/hooks';
import type { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';

interface Props {
  title?: string;
  value?: DropdownOption;
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

export const NamespaceDropdown: React.FC<Props> = ({ title = 'Namespace', value, onSelect, onDeselect, ...props }) => {
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

  return <Dropdown title={title} placeholder='Select namespace' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
