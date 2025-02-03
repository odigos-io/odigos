import React, { useMemo } from 'react';
import { useNamespace } from '@/hooks';
import { Dropdown, type DropdownProps } from '@odigos/ui-components';

interface Props {
  title?: string;
  value?: DropdownProps['options'][0];
  onSelect: (val: DropdownProps['options'][0]) => void;
  onDeselect: (val: DropdownProps['options'][0]) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

export const NamespaceDropdown: React.FC<Props> = ({ title = 'Namespace', value, onSelect, onDeselect, ...props }) => {
  const { allNamespaces } = useNamespace();

  const options = useMemo(() => {
    const payload: DropdownProps['options'] = [];

    allNamespaces?.forEach(({ name: id }) => {
      if (!payload.find((opt) => opt.id === id)) {
        payload.push({ id, value: id });
      }
    });

    return payload;
  }, [allNamespaces]);

  return <Dropdown title={title} placeholder='Select namespace' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} {...props} />;
};
