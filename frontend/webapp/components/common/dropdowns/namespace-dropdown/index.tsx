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

  const namespaceOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    allNamespaces?.forEach(({ name: id }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value: id });
    });

    return options;
  }, [allNamespaces]);

  return <Dropdown title='Namespace' placeholder='Select namespace' options={namespaceOptions} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};
